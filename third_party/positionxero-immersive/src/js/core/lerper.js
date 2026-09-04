/**
 * Frame-rate-independent smoothing utilities.
 *
 * Lerper: named exponential-approach channels. Every scroll progress that
 * drives the scene passes through one of these instead of ScrollTrigger
 * scrub — target moves instantly, current chases at (10 * multiplier)/s.
 * `velocity` (target - current) doubles as a cheap scroll-speed signal.
 *
 * Animator: named time-based tweens with an S-curve default easing, used
 * for section-change values (kvZoom, worksRotate, loaded…).
 */

export class Lerper {
  constructor() {
    this._map = new Map();
  }

  /** Set target; returns the current smoothed value. */
  set(name, target, multiplier = 1) {
    let ch = this._map.get(name);
    if (!ch) {
      ch = { current: target, target, multiplier, velocity: 0 };
      this._map.set(name, ch);
    } else {
      ch.target = target;
      ch.multiplier = multiplier;
    }
    return ch.current;
  }

  get(name) {
    return this._map.get(name)?.current ?? 0;
  }

  velocity(name) {
    return this._map.get(name)?.velocity ?? 0;
  }

  update(dt) {
    for (const ch of this._map.values()) {
      ch.velocity = ch.target - ch.current;
      ch.current += ch.velocity * Math.min(1, dt * 10 * ch.multiplier);
    }
  }
}

/* S-curve easing (logistic, steepness k) — the default "sigmoid(6)" feel. */
export function sigmoid(k = 6) {
  const c = (1 + Math.exp(-k)) / (1 - Math.exp(-k));
  return (t) => {
    const u = 2 * t - 1;
    const s = (1 - Math.exp(-k * u)) / (1 + Math.exp(-k * u));
    return (s * c + 1) / 2;
  };
}

export const easeOutCubic = (t) => 1 - Math.pow(1 - t, 3);

export class Animator {
  constructor() {
    this._vars = new Map();
  }

  register(name, initValue = 0) {
    this._vars.set(name, {
      value: initValue,
      from: initValue,
      to: initValue,
      t: 1,
      duration: 1,
      ease: sigmoid(6),
      done: null
    });
  }

  /** Animate `name` to `to` over `duration` seconds. Returns a promise. */
  animate(name, to, duration = 1, ease = sigmoid(6)) {
    const v = this._vars.get(name);
    if (!v) throw new Error(`Animator: unknown var ${name}`);
    v.from = v.value;
    v.to = to;
    v.duration = Math.max(duration, 1e-6);
    v.t = 0;
    v.ease = ease;
    return new Promise((res) => { v.done = res; });
  }

  setNow(name, value) {
    const v = this._vars.get(name);
    if (!v) return;
    v.value = v.from = v.to = value;
    v.t = 1;
  }

  get(name) {
    return this._vars.get(name)?.value ?? 0;
  }

  update(dt) {
    for (const v of this._vars.values()) {
      if (v.t >= 1) continue;
      v.t = Math.min(1, v.t + dt / v.duration);
      v.value = v.from + (v.to - v.from) * v.ease(v.t);
      if (v.t >= 1 && v.done) {
        v.done();
        v.done = null;
      }
    }
  }
}
