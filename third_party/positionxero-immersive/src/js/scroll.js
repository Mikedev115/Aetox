import gsap from 'gsap';
import { ScrollTrigger } from 'gsap/ScrollTrigger';

gsap.registerPlugin(ScrollTrigger);

/**
 * Top-page scroll choreography. Everything is scrub/reverse-driven so
 * scrolling back rewinds the site. The scene is driven only through
 * gl.setSection(float) via a guarded bridge — a scene error must never
 * break scrolling.
 */

export function initScroll(gl) {
  const sections = gsap.utils.toArray('[data-top-section]');
  if (!sections.length) return;

  const sceneCall = (fn) => {
    try { fn(); } catch { /* scene errors must not break scrolling */ }
  };

  // --- scene section float: one scrub over the whole page ---
  const order = ['kv', 'works_intro', 'works', 'mission', 'vision', 'service'];
  const indexOf = (el) => {
    const i = order.indexOf(el.dataset.topSection);
    return i < 0 ? 0 : i;
  };

  // Cache section geometry outside the scrub hot path — no layout reads
  // per tick. Remeasured on ScrollTrigger refresh (which covers resize).
  let metrics = [];
  const measure = () => {
    metrics = sections.map((el) => {
      const r = el.getBoundingClientRect();
      return { i: indexOf(el), top: r.top + window.scrollY, height: r.height };
    });
  };
  measure();
  ScrollTrigger.addEventListener('refresh', measure);

  const worksCount = document.querySelectorAll('[data-top-section="works"] .works-item').length;

  const update = () => {
    const mid = window.scrollY + window.innerHeight * 0.5;
    let f = 0;
    let worksP = 0;
    for (const m of metrics) {
      if (m.top <= mid) {
        const within = Math.min(1, Math.max(0, (mid - m.top) / m.height));
        f = m.i + within * 0.999;
      }
      if (m.i === 2 && worksCount) {
        // hold-then-slide: screen k holds centered for most of item k's
        // scroll span, sliding to k+1 only in the last stretch
        const raw = Math.max(0, ((mid - m.top) / m.height) * worksCount);
        const k = Math.min(worksCount - 1, Math.floor(raw));
        const frac = raw - k;
        const u = frac < 0.75 ? 0 : Math.min(1, (frac - 0.75) / 0.25);
        const eased = u * u * (3 - 2 * u);
        worksP = Math.min(worksCount - 1, k + eased);
      }
    }
    sceneCall(() => gl.setSection(f));
    sceneCall(() => gl.setWorksProgress(worksP));
  };

  ScrollTrigger.create({
    start: 0,
    end: 'max',
    scrub: true,
    onUpdate: update
  });
  update();

  // --- left rail active state + HUD visibility ---
  const railItems = gsap.utils.toArray('.rail__item');
  const railFor = {
    kv: 'top', works_intro: 'works', works: 'works',
    mission: 'about', vision: 'vision', service: 'service'
  };
  const newsHud = document.querySelector('.news-hud');
  const scrollHint = document.querySelector('.scroll-hint');
  const heroHuds = document.querySelectorAll('.hud--material, .hud--quaternion, .hud--gizmo');

  sections.forEach((el) => {
    ScrollTrigger.create({
      trigger: el,
      start: 'top center',
      end: 'bottom center',
      onToggle: (self) => {
        if (!self.isActive) return;
        const key = railFor[el.dataset.topSection] || 'top';
        railItems.forEach((r) => r.setAttribute('data-active', String(r.dataset.rail === key)));
        const onKv = el.dataset.topSection === 'kv';
        // inert removes invisible controls/links from the tab order
        for (const hud of [newsHud, ...heroHuds]) {
          if (!hud) continue;
          hud.setAttribute('data-hidden', String(!onKv));
          hud.toggleAttribute('inert', !onKv);
        }
        scrollHint?.setAttribute('data-hidden', String(!onKv));
      }
    });
  });

  // --- reveals (reverse on scroll back) ---
  gsap.utils.toArray('.works-item').forEach((item) => {
    gsap.from(item, {
      y: 40,
      opacity: 0,
      duration: 0.8,
      ease: 'power3.out',
      scrollTrigger: {
        trigger: item,
        start: 'top 82%',
        toggleActions: 'play none none reverse'
      }
    });
  });

  // mission/vision marker highlight — CSS class toggled both ways
  gsap.utils.toArray('.statement__marker').forEach((marker, i) => {
    ScrollTrigger.create({
      trigger: marker,
      start: 'top 68%',
      onEnter: () => setTimeout(() => marker.classList.add('is-lit'), i % 3 * 160),
      onLeaveBack: () => marker.classList.remove('is-lit')
    });
  });

  gsap.utils.toArray('.statement__en, .service-card').forEach((el) => {
    gsap.from(el, {
      y: 28,
      opacity: 0,
      duration: 0.7,
      ease: 'power2.out',
      scrollTrigger: {
        trigger: el,
        start: 'top 80%',
        toggleActions: 'play none none reverse'
      }
    });
  });

  window.addEventListener('resize', debounce(() => {
    sceneCall(() => gl.resize());
    ScrollTrigger.refresh();
  }, 200));
}

function debounce(fn, ms) {
  let id;
  return (...a) => {
    clearTimeout(id);
    id = setTimeout(() => fn(...a), ms);
  };
}
