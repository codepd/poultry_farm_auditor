import React, { createContext, useContext, useState, useEffect, useCallback, useRef, ReactNode } from 'react';
import { AppLanguage, translations, translateItemName, translateStageTag } from '../locales/translations';
import { useAuth } from './AuthContext';
import { usersAPI } from '../services/api';

interface LanguageContextType {
  language: AppLanguage;
  setLanguage: (lang: AppLanguage) => Promise<void>;
  t: (key: string, vars?: Record<string, string | number>) => string;
  tItem: (itemName?: string | null) => string;
  tStage: (stage?: string) => string;
  formatMonthYear: (year: number, month: number, shortMonth?: boolean) => string;
}

const LanguageContext = createContext<LanguageContextType | undefined>(undefined);

const STORAGE_KEY = 'kolipannai_lang';

function normalizeLang(val?: string | null): AppLanguage {
  if (val && val.toLowerCase().startsWith('ta')) return 'ta';
  return 'en';
}

export const LanguageProvider: React.FC<{ children: ReactNode }> = ({ children }) => {
  const { user, isAuthenticated, updateUser } = useAuth();
  const manualOverrideRef = useRef(false);
  const [language, setLanguageState] = useState<AppLanguage>(() => {
    const saved = localStorage.getItem(STORAGE_KEY);
    if (saved === 'ta' || saved === 'en') return saved;
    if (user?.preferred_language === 'ta' || user?.preferred_language === 'en') {
      return user.preferred_language;
    }
    return 'en';
  });

  // Sync from user profile on login or session bootstrap across devices
  useEffect(() => {
    if (!isAuthenticated || !user) return;

    if (!manualOverrideRef.current && (user.preferred_language === 'ta' || user.preferred_language === 'en')) {
      setLanguageState(user.preferred_language);
      localStorage.setItem(STORAGE_KEY, user.preferred_language);
    }

    // Fetch latest profile from server so cross-device preference changes are reflected automatically
    let cancelled = false;
    usersAPI
      .getProfile()
      .then((profile) => {
        if (cancelled || !profile || manualOverrideRef.current) return;
        const serverLang = normalizeLang(profile.preferred_language);
        setLanguageState(serverLang);
        localStorage.setItem(STORAGE_KEY, serverLang);
        if (user.preferred_language !== serverLang || (profile.full_name && user.full_name !== profile.full_name)) {
          updateUser({
            preferred_language: serverLang,
            ...(profile.full_name ? { full_name: profile.full_name } : {}),
          });
        }
      })
      .catch(() => {
        // Ignore offline/network errors on background profile sync
      });

    return () => {
      cancelled = true;
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [isAuthenticated, user?.user_id]);

  // Update <html lang="..."> attribute for font rendering
  useEffect(() => {
    document.documentElement.lang = language === 'ta' ? 'ta' : 'en';
  }, [language]);

  const setLanguage = useCallback(
    async (lang: AppLanguage) => {
      manualOverrideRef.current = true;
      setLanguageState(lang);
      localStorage.setItem(STORAGE_KEY, lang);
      if (isAuthenticated) {
        updateUser({ preferred_language: lang });
        try {
          await usersAPI.updateProfile({ preferred_language: lang });
        } catch (err) {
          console.error('Failed to persist language preference to user profile:', err);
        }
      }
    },
    [isAuthenticated, updateUser]
  );

  const t = useCallback(
    (key: string, vars?: Record<string, string | number>): string => {
      const dict = translations[language] || translations.en;
      let text = dict[key] ?? translations.en[key] ?? key;
      if (vars) {
        Object.entries(vars).forEach(([k, v]) => {
          text = text.replace(new RegExp(`\\{${k}\\}`, 'g'), String(v));
        });
      }
      return text;
    },
    [language]
  );

  const tItem = useCallback(
    (itemName?: string | null) => translateItemName(itemName, language),
    [language]
  );

  const tStage = useCallback(
    (stage?: string) => translateStageTag(stage, language),
    [language]
  );

  const formatMonthYear = useCallback(
    (year: number, month: number, shortMonth = false) => {
      const d = new Date(year, month - 1, 1);
      const locale = language === 'ta' ? 'ta-IN' : 'en-IN';
      return d.toLocaleString(locale, {
        month: shortMonth ? 'short' : 'long',
        year: 'numeric',
      });
    },
    [language]
  );

  return (
    <LanguageContext.Provider
      value={{
        language,
        setLanguage,
        t,
        tItem,
        tStage,
        formatMonthYear,
      }}
    >
      {children}
    </LanguageContext.Provider>
  );
};

export const useLanguage = () => {
  const ctx = useContext(LanguageContext);
  if (!ctx) {
    throw new Error('useLanguage must be used within a LanguageProvider');
  }
  return ctx;
};
