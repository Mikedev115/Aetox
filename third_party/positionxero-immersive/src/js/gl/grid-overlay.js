import * as THREE from 'three';
import { GLSL_CONSTANTS, GLSL_ROTATE2 } from './shader-lib.js';

/**
 * The fine grid + registration crosses drawn over the LED wall (light
 * variant, bent to the same cylinder) and over the light mission/vision
 * scene (dark variant, flat).
 */

const vert = /* glsl */ `
${GLSL_CONSTANTS}
${GLSL_ROTATE2}
uniform vec3 uScale;
uniform float uCylinder; // 1 = bend to the wall cylinder, 0 = flat plane
varying vec2 vUv;
void main() {
  vec3 pos = position;
  vec3 out_;
  if (uCylinder > 0.5) {
    float theta = pos.x * PI;
    out_ = vec3(
      sin(theta) * 0.5 * uScale.x,
      pos.y * uScale.y * 1.5,
      -cos(theta) * uScale.x * 0.5
    );
    // pull just in front of the wall so lines are not z-fighting
    out_ *= 0.985;
  } else {
    out_ = vec3(pos.x * uScale.x, pos.y * uScale.y, 0.0);
  }
  gl_Position = projectionMatrix * modelViewMatrix * vec4(out_, 1.0);
  vUv = uv;
}
`;

const frag = /* glsl */ `
uniform vec2 uGrid;
uniform float uScroll;
uniform vec3 uColor;
uniform float uAlpha;
varying vec2 vUv;
void main() {
  vec2 g = vUv * uGrid;
  g.y -= uScroll * 1.5;
  vec2 cell = fract(g);
  float lineX = smoothstep(0.46, 0.5, abs(cell.x - 0.5));
  float lineY = smoothstep(0.46, 0.5, abs(cell.y - 0.5));
  float line = max(lineX, lineY);
  gl_FragColor = vec4(uColor, uAlpha * line * 0.2);
}
`;

const crossFrag = /* glsl */ `
uniform vec3 uColor;
uniform float uScroll;
varying vec2 vUv;
void main() {
  vec2 p = fract(vUv * 9.0 + vec2(0.0, uScroll * 0.1)) - 0.5;
  float armH = smoothstep(0.003, 0.0015, abs(p.y)) * step(abs(p.x), 0.028);
  float armV = smoothstep(0.003, 0.0015, abs(p.x)) * step(abs(p.y), 0.028);
  float cross_ = max(armH, armV);
  gl_FragColor = vec4(uColor, cross_ * 0.3);
}
`;

export class GridOverlay {
  constructor({ dark = false, cylinder = true } = {}) {
    this.group = new THREE.Group();
    const color = dark ? new THREE.Color(0, 0, 0) : new THREE.Color(1, 1, 1);

    this.uniforms = {
      uScale: { value: new THREE.Vector3(14.8, 14.8, 1) },
      uCylinder: { value: cylinder ? 1 : 0 },
      uGrid: { value: new THREE.Vector2(64, 64) },
      uScroll: { value: 0 },
      uColor: { value: color },
      // the dark grid over the light scene is much fainter on the reference
      uAlpha: { value: dark ? 0.1 : 0.5 }
    };

    const lines = new THREE.Mesh(
      new THREE.PlaneGeometry(1, 1, 64, 64),
      new THREE.ShaderMaterial({
        vertexShader: vert,
        fragmentShader: frag,
        uniforms: this.uniforms,
        transparent: true,
        depthWrite: false,
        depthTest: false,
        side: THREE.DoubleSide
      })
    );
    lines.renderOrder = -999;
    lines.frustumCulled = false;
    this.group.add(lines);

    const crosses = new THREE.Mesh(
      new THREE.PlaneGeometry(1, 1, 8, 8),
      new THREE.ShaderMaterial({
        vertexShader: vert,
        fragmentShader: crossFrag,
        uniforms: this.uniforms,
        transparent: true,
        depthWrite: false,
        depthTest: false,
        side: THREE.DoubleSide
      })
    );
    crosses.renderOrder = -998;
    crosses.frustumCulled = false;
    this.group.add(crosses);
  }

  resize(camera) {
    const h = 2 * Math.abs(camera.position.z) * Math.tan(THREE.MathUtils.degToRad(camera.fov) / 2);
    const w = Math.max(h, h * camera.aspect);
    this.uniforms.uScale.value.set(w, w, 1);
  }

  setScroll(v) {
    this.uniforms.uScroll.value = v;
  }

  dispose() {
    for (const m of this.group.children) {
      m.geometry.dispose();
      m.material.dispose();
    }
  }
}
