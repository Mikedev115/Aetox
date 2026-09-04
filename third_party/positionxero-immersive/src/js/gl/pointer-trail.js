import * as THREE from 'three';
import { ProceduralTexture } from './procedural-texture.js';

/**
 * A cheap pointer-velocity buffer standing in for the reference site's
 * fluid sim: pointer movement splats velocity (RG channels) into a small
 * ping-pong texture that dissipates each frame. The wall's tile sides
 * glow where |velocity| is high, and the mixer wobbles UVs with it.
 */

const frag = /* glsl */ `
uniform sampler2D uPrev;
uniform vec2 uPointer;      // 0..1
uniform vec2 uVelocity;     // NDC per frame
uniform float uAspect;
varying vec2 vUv;
void main() {
  vec4 prev = texture2D(uPrev, vUv) * 0.965;  // lingering dissipation
  vec2 d = vUv - uPointer;
  d.x *= uAspect;
  float splat = exp(-dot(d, d) * 70.0);       // wide wake around the cursor
  vec2 vel = prev.xy + uVelocity * splat * 6.0;
  gl_FragColor = vec4(vel, 0.0, 1.0);
}
`;

export class PointerTrail {
  constructor(renderer) {
    this.renderer = renderer;
    this.a = new ProceduralTexture(renderer, {
      width: 128, height: 128, fragmentShader: frag,
      uniforms: {
        uPrev: { value: null },
        uPointer: { value: new THREE.Vector2(0.5, 0.5) },
        uVelocity: { value: new THREE.Vector2() },
        uAspect: { value: 1 }
      },
      wrap: THREE.ClampToEdgeWrapping
    });
    this.b = new ProceduralTexture(renderer, {
      width: 128, height: 128, fragmentShader: frag,
      uniforms: this.a.uniforms,
      wrap: THREE.ClampToEdgeWrapping
    });
    this._flip = false;
    this._pointer = new THREE.Vector2(0.5, 0.5);
    this._last = new THREE.Vector2(0.5, 0.5);
  }

  get texture() {
    return (this._flip ? this.a : this.b).texture;
  }

  setPointer(x01, y01) {
    this._pointer.set(x01, y01);
  }

  setAspect(a) {
    this.a.uniforms.uAspect.value = a;
  }

  update() {
    const u = this.a.uniforms;
    u.uPointer.value.copy(this._pointer);
    u.uVelocity.value.set(
      this._pointer.x - this._last.x,
      this._pointer.y - this._last.y
    );
    this._last.copy(this._pointer);

    const src = this._flip ? this.a : this.b;
    const dst = this._flip ? this.b : this.a;
    u.uPrev.value = src.texture;
    dst.render();
    this._flip = !this._flip;
  }

  dispose() {
    this.a.dispose();
    this.b.dispose();
  }
}
