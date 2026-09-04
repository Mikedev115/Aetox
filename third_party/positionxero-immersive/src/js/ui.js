/**
 * Shared header wiring: hamburger / side menu / sound toggle.
 * Used by every page.
 */

export function initHeader() {
  const burger = document.querySelector('.hamburger');
  const backdrop = document.querySelector('.side-menu__backdrop');

  const setOpen = (open) => {
    document.body.setAttribute('data-menu-open', String(open));
    burger?.setAttribute('aria-expanded', String(open));
  };

  burger?.addEventListener('click', () => {
    setOpen(document.body.getAttribute('data-menu-open') !== 'true');
  });
  backdrop?.addEventListener('click', () => setOpen(false));
  document.addEventListener('keydown', (e) => {
    if (e.key === 'Escape') setOpen(false);
  });
  document.querySelectorAll('.side-menu__nav a').forEach((a) =>
    a.addEventListener('click', () => setOpen(false))
  );

  // Sound toggle is visual-only in this rebuild (no audio track shipped).
  const sound = document.querySelector('.sound-toggle');
  sound?.addEventListener('click', () => {
    const muted = sound.getAttribute('data-muted') !== 'true';
    sound.setAttribute('data-muted', String(muted));
    sound.setAttribute('aria-pressed', String(!muted));
  });
}
