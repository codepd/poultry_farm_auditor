#!/bin/zsh
# Open a URL in the Chrome profile named "codepd".
# In Chrome's profile list that name is directory "Profile 8"
# (codepd3@gmail.com). The folder literally named "codepd" is a different
# profile, "Your Chrome". Override with CHROME_PROFILE if this moves.
set -euo pipefail

profile="${CHROME_PROFILE:-Profile 8}"
url="${1:-https://d-9f676c1851.awsapps.com/start}"
chrome="/Applications/Google Chrome.app/Contents/MacOS/Google Chrome"

if [[ ! -x "$chrome" ]]; then
  echo "Google Chrome not found at: $chrome" >&2
  exit 1
fi

# Invoke the Chrome binary directly and return at once. `open -n` starts a
# second Chrome instance, which cannot use a profile that is already open and
# shows up as a stray window. A second launch of this binary hands the URL to
# the running "codepd" window.
"$chrome" --profile-directory="$profile" "$url" >/dev/null 2>&1 &
exit 0
