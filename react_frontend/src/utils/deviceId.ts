const STORAGE_KEY = 'app_device_id';

/**
 * Stable per-browser profile id for X-Device-Id (not a secret; server stores with auth session).
 */
export function getOrCreateDeviceId(): string {
  try {
    let id = localStorage.getItem(STORAGE_KEY);
    if (id && id.trim()) {
      return id.trim();
    }
    id = crypto.randomUUID();
    localStorage.setItem(STORAGE_KEY, id);
    return id;
  } catch {
    // localStorage unavailable (e.g. private mode quirks) — still send a per-tab value
    return `ephemeral-${crypto.randomUUID()}`;
  }
}
