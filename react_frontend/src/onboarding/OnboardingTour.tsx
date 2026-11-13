import React, { useCallback, useEffect, useLayoutEffect, useState } from 'react';
import { useAuth } from '../context/AuthContext';
import { useLanguage } from '../context/LanguageContext';
import { usersAPI } from '../services/api';
import { CURRENT_TOUR_VERSION, getPendingSteps, TOUR_STEPS, TourStep } from './tourSteps';
import './OnboardingTour.css';

const SNOOZE_KEY = 'kolipannai_tour_snoozed';
const progressKey = (userId?: number) => `kolipannai_tour_progress_${userId ?? 'anon'}`;
export const OPEN_TOUR_EVENT = 'openOnboardingTour';

type Rect = { top: number; left: number; width: number; height: number };

const findTarget = (step?: TourStep): HTMLElement | null =>
  step?.target ? (document.querySelector(`[data-tour="${step.target}"]`) as HTMLElement | null) : null;

const OnboardingTour: React.FC = () => {
  const { user, isAuthenticated } = useAuth();
  const { t } = useLanguage();
  const [steps, setSteps] = useState<TourStep[]>([]);
  const [index, setIndex] = useState(0);
  const [isWhatsNew, setIsWhatsNew] = useState(false);
  const [isReplay, setIsReplay] = useState(false);
  const [rect, setRect] = useState<Rect | null>(null);

  const open = steps.length > 0;
  const step = steps[index];

  // Decide whether to show the tour after login (skipped while the forced password modal is up)
  useEffect(() => {
    if (!isAuthenticated || !user || user.must_set_password) return;
    if (sessionStorage.getItem(SNOOZE_KEY) === '1') return;

    let cancelled = false;
    usersAPI
      .getProfile()
      .then((profile) => {
        if (cancelled) return;
        const completed = Number(profile?.onboarding_version ?? 0);
        const pending = getPendingSteps(completed);
        if (pending.length === 0) return;

        let resumeIdx = 0;
        try {
          const saved = JSON.parse(localStorage.getItem(progressKey(user.user_id)) || 'null');
          if (saved && saved.version === CURRENT_TOUR_VERSION && saved.index < pending.length) {
            resumeIdx = saved.index;
          }
        } catch {
          // ignore corrupt progress
        }
        setIsWhatsNew(completed > 0);
        setIsReplay(false);
        setSteps(pending);
        setIndex(resumeIdx);
      })
      .catch(() => {});
    return () => {
      cancelled = true;
    };
  }, [isAuthenticated, user?.user_id, user?.must_set_password]); // eslint-disable-line react-hooks/exhaustive-deps

  // Manual replay from the profile menu
  useEffect(() => {
    const onOpen = () => {
      setIsWhatsNew(false);
      setIsReplay(true);
      setSteps(TOUR_STEPS);
      setIndex(0);
    };
    window.addEventListener(OPEN_TOUR_EVENT, onOpen);
    return () => window.removeEventListener(OPEN_TOUR_EVENT, onOpen);
  }, []);

  // Persist progress so the user resumes where they stopped (until every step is seen)
  useEffect(() => {
    if (!open || isReplay || !user) return;
    localStorage.setItem(progressKey(user.user_id), JSON.stringify({ version: CURRENT_TOUR_VERSION, index }));
  }, [open, index, isReplay, user]);

  // Locate & highlight the target; wait briefly for async content, skip steps whose target never appears
  useLayoutEffect(() => {
    if (!open || !step) return;
    if (!step.target) {
      setRect(null);
      return;
    }
    let attempts = 0;
    let timer: ReturnType<typeof setTimeout>;
    const locate = () => {
      const el = findTarget(step);
      if (el) {
        el.scrollIntoView?.({ block: 'center', behavior: 'smooth' });
        const r = el.getBoundingClientRect();
        setRect({ top: r.top, left: r.left, width: r.width, height: r.height });
        return;
      }
      attempts += 1;
      if (attempts < 10) {
        timer = setTimeout(locate, 200);
      } else {
        // Not available for this user/device (e.g. viewer role, app already installed)
        setSteps((prev) => prev.filter((s) => s.id !== step.id));
        setIndex((i) => Math.min(i, Math.max(0, steps.length - 2)));
      }
    };
    locate();
    return () => clearTimeout(timer);
  }, [open, step, steps.length]);

  // Keep highlight aligned on scroll / resize
  useEffect(() => {
    if (!open || !step?.target) return;
    const update = () => {
      const el = findTarget(step);
      if (!el) return;
      const r = el.getBoundingClientRect();
      setRect({ top: r.top, left: r.left, width: r.width, height: r.height });
    };
    window.addEventListener('resize', update);
    window.addEventListener('scroll', update, true);
    return () => {
      window.removeEventListener('resize', update);
      window.removeEventListener('scroll', update, true);
    };
  }, [open, step]);

  const close = useCallback(() => {
    setSteps([]);
    setIndex(0);
    setRect(null);
  }, []);

  const finish = useCallback(async () => {
    if (!isReplay && user) {
      localStorage.removeItem(progressKey(user.user_id));
      try {
        await usersAPI.updateProfile({ onboarding_version: CURRENT_TOUR_VERSION });
      } catch {
        // Will be offered again next session if the save failed
      }
    }
    close();
  }, [isReplay, user, close]);

  const later = useCallback(() => {
    if (!isReplay) sessionStorage.setItem(SNOOZE_KEY, '1');
    close();
  }, [isReplay, close]);

  if (!open || !step) return null;

  const isLast = index === steps.length - 1;
  const pad = 6;
  const popoverStyle: React.CSSProperties = rect
    ? (() => {
        const below = rect.top + rect.height + 12;
        const fitsBelow = below + 200 < window.innerHeight;
        const left = Math.max(12, Math.min(rect.left, window.innerWidth - 352));
        return fitsBelow ? { top: below, left } : { top: Math.max(12, rect.top - 212), left };
      })()
    : {};

  return (
    <div className="tour-root" data-testid="onboarding-tour" role="dialog" aria-modal="true">
      {rect ? (
        <div
          className="tour-spotlight"
          style={{
            top: rect.top - pad,
            left: rect.left - pad,
            width: rect.width + pad * 2,
            height: rect.height + pad * 2,
          }}
        />
      ) : (
        <div className="tour-backdrop" />
      )}
      <div className={`tour-popover ${rect ? '' : 'centered'}`} style={popoverStyle}>
        {isWhatsNew && <div className="tour-badge">{t('tour.whatsNew')}</div>}
        <div className="tour-step-count">{t('tour.stepOf', { current: index + 1, total: steps.length })}</div>
        <h4 className="tour-title">{t(step.titleKey)}</h4>
        <p className="tour-body">{t(step.bodyKey)}</p>
        <div className="tour-actions">
          <button type="button" className="tour-link" onClick={later}>
            {t('tour.later')}
          </button>
          <div className="tour-nav">
            {index > 0 && (
              <button type="button" className="tour-btn secondary" onClick={() => setIndex(index - 1)}>
                {t('tour.back')}
              </button>
            )}
            {isLast ? (
              <button type="button" className="tour-btn" onClick={finish}>
                {t('tour.finish')}
              </button>
            ) : (
              <button type="button" className="tour-btn" onClick={() => setIndex(index + 1)}>
                {t('tour.next')}
              </button>
            )}
          </div>
        </div>
      </div>
    </div>
  );
};

export default OnboardingTour;
