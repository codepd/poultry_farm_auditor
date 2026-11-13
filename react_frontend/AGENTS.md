# Frontend rules

## Keep the onboarding walkthrough in sync with every user-facing change

Whenever you add or materially change a user-visible feature (new page, button, workflow, profit view, etc.):

1. Update `src/onboarding/tourSteps.ts` — add a step or edit the existing one, and set its `version` to the current max version + 1 (never reuse/lower versions).
2. Add the step's `titleKey` / `bodyKey` to **both** `en` and `ta` in `src/locales/translations.ts`.
3. Put `data-tour="<target>"` on the element the step highlights.
4. Run `npm test` — the "Onboarding tour" tests fail if versions, translations, or targets are inconsistent.

Users who already completed the tour then automatically see only the new/changed steps as "What's new".
