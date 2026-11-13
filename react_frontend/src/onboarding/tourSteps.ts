/**
 * Guided walkthrough (onboarding tour) definition.
 *
 * HOW TO KEEP THE TOUR UP TO DATE (required for every user-facing change):
 *  1. Add a step (or edit an existing one) below and give it `version: CURRENT + 1`.
 *  2. Add its `titleKey` / `bodyKey` to BOTH `en` and `ta` in `locales/translations.ts`.
 *  3. Put `data-tour="<target>"` on the element the step should highlight.
 *
 * Users who already finished the tour will automatically see ONLY the steps whose
 * `version` is greater than the version stored on their profile (`users.onboarding_version`),
 * shown as "What's new". New users see every step.
 */
export interface TourStep {
  id: string;
  /** Tour version in which this step was added or last materially changed. */
  version: number;
  /** Value of the `data-tour` attribute to highlight. Omit for a centered intro card. */
  target?: string;
  titleKey: string;
  bodyKey: string;
}

export const TOUR_STEPS: TourStep[] = [
  { id: 'welcome', version: 1, titleKey: 'tour.welcome.title', bodyKey: 'tour.welcome.body' },
  { id: 'language', version: 1, target: 'language', titleKey: 'tour.language.title', bodyKey: 'tour.language.body' },
  { id: 'nav', version: 1, target: 'nav', titleKey: 'tour.nav.title', bodyKey: 'tour.nav.body' },
  { id: 'month', version: 1, target: 'month', titleKey: 'tour.month.title', bodyKey: 'tour.month.body' },
  { id: 'presets', version: 1, target: 'profit-presets', titleKey: 'tour.presets.title', bodyKey: 'tour.presets.body' },
  { id: 'share', version: 1, target: 'share-report', titleKey: 'tour.share.title', bodyKey: 'tour.share.body' },
  { id: 'quick-log', version: 1, target: 'quick-log', titleKey: 'tour.quickLog.title', bodyKey: 'tour.quickLog.body' },
  { id: 'install', version: 1, target: 'install', titleKey: 'tour.install.title', bodyKey: 'tour.install.body' },
];

export const CURRENT_TOUR_VERSION = TOUR_STEPS.reduce((max, s) => Math.max(max, s.version), 0);

/** Steps the user has not seen yet. `completedVersion = 0` means a brand-new user (full tour). */
export function getPendingSteps(completedVersion: number): TourStep[] {
  return TOUR_STEPS.filter((s) => s.version > completedVersion);
}
