import gsap from 'gsap';
import { ScrollTrigger } from 'gsap/ScrollTrigger';
import Lenis from 'lenis';

gsap.registerPlugin(ScrollTrigger);

/**
 * Scroll manager: Lenis smooth scroll (defaults, lerp 0.1) + plain
 * ScrollTriggers used purely as progress trackers (no scrub, no pin).
 * The GL scene reads the raw progress each frame and smooths it itself.
 *
 * Snap points:
 *  - works: 1/(N-1)
 *  - mission/calc/vision snapper (works_outro..vision): ratio-derived array
 *  - vision_service wrap: [0, 1]
 *  - service: [0, 0.4]
 *  - cognexa: [0.9]
 */

export class ScrollManager {
  constructor({ onSectionChange } = {}) {
    this.lenis = new Lenis({});
    this.onSectionChange = onSectionChange;
    this.section = 'kv';
    this.triggers = new Map();
    this.progress = {
      worksTitle: 0, worksProgress: 0, worksOutro: 0, missionIn: 0,
      calcIn: 0, vision: 0, visionOut: 0, serviceIn: 0, serviceProgress: 0,
      cognexaIn: 0, cognexa: 0, missionRaw: 0
    };

    // lenis is driven from the main rAF in main.js via update(t);
    // keep ScrollTrigger in lockstep with lenis' smoothed scroll values
    this.lenis.on('scroll', ScrollTrigger.update);
  }

  update(timeMs) {
    this.lenis.raf(timeMs);
  }

  get velocity() {
    return this.lenis.velocity;
  }

  get scroll() {
    return this.lenis.scroll;
  }

  _enterSection(name) {
    if (this.section === name) return;
    this.section = name;
    document.body.setAttribute('data-section', name);
    this.onSectionChange?.(name);
  }

  init() {
    const q = (sel) => document.querySelector(sel);
    const sec = (name) => q(`[data-top-section="${name}"]`);

    const reg = (key, vars) => {
      const trig = ScrollTrigger.create(vars);
      this.triggers.set(key, trig);
      return trig;
    };

    const enter = (name) => ({
      onEnter: () => this._enterSection(name),
      onEnterBack: () => this._enterSection(name)
    });

    /* --- sections --- */
    reg('kv', {
      trigger: sec('kv'), start: 'top bottom', end: 'bottom bottom-=100px', ...enter('kv')
    });
    reg('works_intro', {
      trigger: sec('works_intro'), start: 'top bottom', end: 'bottom bottom',
      onUpdate: (self) => { this.progress.worksTitle = self.progress; },
      ...enter('works_intro')
    });

    const worksEl = sec('works');
    const worksItems = gsap.utils.toArray('.works-scroll__item');
    reg('works_snap', {
      trigger: worksEl, start: 'top top', end: 'bottom bottom',
      snap: { snapTo: 1 / (worksItems.length - 1), duration: 1, directional: false }
    });
    reg('works_event', {
      trigger: worksEl, start: 'top bottom', end: 'bottom bottom', ...enter('works')
    });
    reg('works_progress', {
      trigger: worksEl, start: 'top bottom', end: 'bottom top',
      onUpdate: (self) => { this.progress.worksProgress = self.progress; }
    });
    worksItems.forEach((el, i) => {
      reg(`works_item_${i}`, {
        trigger: el, start: 'top center', end: 'bottom center',
        onEnter: () => this._emitItem('works', i, true),
        onEnterBack: () => this._emitItem('works', i, true),
        onLeave: () => this._emitItem('works', i, false),
        onLeaveBack: () => this._emitItem('works', i, false)
      });
    });

    reg('works_outro', {
      trigger: sec('works_outro'), start: 'top bottom', end: 'bottom bottom',
      onUpdate: (self) => { this.progress.worksOutro = self.progress; },
      ...enter('works_outro')
    });
    reg('mission_in', {
      trigger: sec('mission_in'), start: 'top bottom', end: 'top top',
      onUpdate: (self) => { this.progress.missionIn = self.progress; },
      ...enter('mission_in')
    });
    reg('mission', {
      trigger: sec('mission'), start: 'top bottom', end: 'bottom bottom',
      onUpdate: (self) => { this.progress.missionRaw = self.progress; },
      ...enter('mission')
    });
    reg('calc', {
      trigger: sec('calc'), start: 'top bottom', end: 'top top',
      onUpdate: (self) => { this.progress.calcIn = self.progress; },
      ...enter('calc')
    });
    reg('vision', {
      trigger: sec('vision'), start: 'top bottom', end: 'top top',
      onUpdate: (self) => { this.progress.vision = self.progress; },
      ...enter('vision')
    });
    reg('vision_out', {
      trigger: sec('vision_out'), start: 'top bottom', end: 'top top',
      onUpdate: (self) => { this.progress.visionOut = self.progress; },
      ...enter('vision_out')
    });

    // snapper across works_outro..vision. The array is positional and must
    // match the runway heights in main.css —
    // 140lvh / 100lvh / 210vh / 260vh / 180vh.
    const snapper = [sec('works_outro'), sec('mission_in'), sec('mission'), sec('calc'), sec('vision')];
    if (snapper.every(Boolean)) {
      const ratios = [1.4, 1, 2.1, 2.6, 1.8];
      const total = ratios.reduce((a, b) => a + b, 0) - 1;
      const points = [0.5 / total];
      let acc = 0;
      for (const r of ratios) {
        acc += r;
        points.push(acc / total);
      }
      const wrap = this._wrapRange(snapper[0], snapper[snapper.length - 1]);
      reg('mv_snapper', {
        trigger: wrap.first, endTrigger: wrap.last,
        start: 'top top', end: 'bottom bottom',
        snap: { snapTo: points.filter((p) => p <= 1), duration: 1, directional: false }
      });
    }

    const vsWrapFirst = sec('vision_out');
    const vsWrapLast = sec('service_in');
    reg('vision_service_wrap', {
      trigger: vsWrapFirst, endTrigger: vsWrapLast,
      start: 'top bottom', end: 'bottom top',
      snap: { snapTo: [0, 1], duration: 1, directional: false }
    });

    reg('service_in', {
      trigger: sec('service_in'), start: 'top bottom+=150%', end: 'top center',
      onUpdate: (self) => { this.progress.serviceIn = self.progress; },
      ...enter('service_in')
    });

    const serviceEl = sec('service');
    reg('service_snap', {
      trigger: serviceEl, start: 'top top', end: 'bottom bottom',
      snap: { snapTo: [0, 0.4], duration: 1, directional: false }
    });
    reg('service_progress', {
      trigger: serviceEl, start: 'top bottom', end: 'bottom top',
      onUpdate: (self) => { this.progress.serviceProgress = self.progress; }
    });
    gsap.utils.toArray('.service-scroll__item').forEach((el, i) => {
      const name = i === 2 ? 'cognexa_item' : `service_${i}`;
      reg(`service_item_${i}`, {
        trigger: el, start: 'top bottom', end: 'bottom bottom',
        onEnter: () => {
          this._emitItem('service', i, true);
          this._enterSection(i === 2 ? 'cognexa' : 'service');
        },
        onEnterBack: () => {
          this._emitItem('service', i, true);
          this._enterSection(i === 2 ? 'cognexa' : 'service');
        },
        onLeave: () => this._emitItem('service', i, false),
        onLeaveBack: () => this._emitItem('service', i, false)
      });
      if (i === 2) {
        reg('cognexa_in', {
          trigger: el, start: 'top bottom', end: 'bottom bottom',
          onUpdate: (self) => { this.progress.cognexaIn = self.progress; }
        });
      }
    });

    const cognexaEl = sec('cognexa');
    reg('cognexa', {
      trigger: cognexaEl, start: 'top bottom', end: 'bottom bottom',
      snap: { snapTo: [0.9], duration: 1, directional: false },
      onUpdate: (self) => { this.progress.cognexa = self.progress; },
      onEnterBack: () => this._enterSection('cognexa'),
      onLeave: () => this._enterSection('footer')
    });

    ScrollTrigger.refresh();
  }

  _wrapRange(first, last) {
    return { first, last };
  }

  _itemHandlers = { works: [], service: [] };

  onItem(kind, fn) {
    this._itemHandlers[kind].push(fn);
  }

  _emitItem(kind, i, entered) {
    for (const fn of this._itemHandlers[kind]) fn(i, entered);
  }

  scrollTo(target, opts = {}) {
    this.lenis.scrollTo(target, {
      duration: 1.2,
      easing: (t) => Math.min(1, 1.001 - Math.pow(2, -10 * t)),
      ...opts
    });
  }

  destroy() {
    for (const t of this.triggers.values()) t.kill();
    this.lenis.destroy();
  }
}
