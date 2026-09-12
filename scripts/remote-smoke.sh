#!/usr/bin/env sh
# The phase-3 smoke: a Linux machine as its own remote host, reached the
# way the desktop reaches one — the real ssh, the real sshd, the scripts in
# internal/engine/remote run by the host's real sh.
#
#   scripts/remote-smoke.sh              # this machine as the host (needs sshd, sudo to start it)
#   scripts/remote-smoke.sh user@host    # a host of your own (needs a key in its authorized_keys)
#
# With no argument it makes this machine reachable as $USER@localhost: a
# throwaway key in ~/.ssh, added to authorized_keys, sshd started if it is
# not. That is what the CI Linux job runs. It leaves the key behind on
# purpose — a CI runner is thrown away, and on a machine of your own you can
# see exactly what was added (the comment on the key names this script).
set -eu

cd "$(dirname "$0")/.."
target="${1:-}"

if [ -z "$target" ]; then
  if [ "$(uname -s)" != "Linux" ]; then
    echo "remote-smoke: without a host argument this needs a Linux machine to be its own host" >&2
    exit 2
  fi
  mkdir -p ~/.ssh && chmod 700 ~/.ssh
  key=~/.ssh/aetox-remote-smoke
  if [ ! -f "$key" ]; then
    ssh-keygen -q -t ed25519 -N '' -C 'aetox scripts/remote-smoke.sh' -f "$key"
  fi
  touch ~/.ssh/authorized_keys && chmod 600 ~/.ssh/authorized_keys
  grep -qF "$(cat "$key.pub")" ~/.ssh/authorized_keys || cat "$key.pub" >> ~/.ssh/authorized_keys
  # The key is the one ssh reaches for: an IdentityFile line in ssh_config
  # for localhost, which is also how a person would set this up by hand.
  touch ~/.ssh/config && chmod 600 ~/.ssh/config
  if ! grep -q 'aetox-remote-smoke' ~/.ssh/config; then
    printf '\nHost localhost\n  IdentityFile %s\n  IdentitiesOnly yes\n' "$key" >> ~/.ssh/config
  fi
  if ! pgrep -x sshd >/dev/null 2>&1; then
    if command -v systemctl >/dev/null 2>&1; then
      sudo systemctl start ssh 2>/dev/null || sudo systemctl start sshd
    else
      sudo service ssh start 2>/dev/null || sudo service sshd start
    fi
  fi
  target="$(id -un)@localhost"
fi

echo "remote-smoke: host is $target"
AETOX_REMOTE_SMOKE="$target" go test -count=1 -v -timeout 10m ./internal/engine/remote -run TestRemoteSmoke
