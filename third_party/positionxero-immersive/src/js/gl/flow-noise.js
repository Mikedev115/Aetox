import * as THREE from 'three';
import { ProceduralTexture } from './procedural-texture.js';
import { GLSL_SIMPLEX3 } from './shader-lib.js';

/**
 * The shared 64×64 animated flow-noise texture.
 * Four channels of domain-warped simplex noise:
 *   rgb — three offset fast channels (feed the wall colour phases and the
 *         glass roughness mask)
 *   a   — one slower, differently-warped channel (feeds the zebra bands)
 * Re-rendered every frame so everything that samples it "crawls".
 */

const frag = /* glsl */ `
${GLSL_SIMPLEX3}
uniform float uTime;
uniform float uScreenAspectRatio;
varying vec2 vUv;

void main() {
  float t = uTime * 0.1;

  vec2 p = vUv * vec2(uScreenAspectRatio, 1.0) - 0.5;
  vec2 q = p * 0.5;

  // low-frequency warp field
  float wx = simplex3(vec3(q + 1234.0, t));
  float wy = simplex3(vec3(q + 5678.0, t + 10.0));
  vec2 warped = p * 0.6 + vec2(wx, wy) * 0.7;

  vec4 col;
  col.r = simplex3(vec3(warped + 1.0, t));
  col.g = simplex3(vec3(warped + 2.0, t + 1.0));
  col.b = simplex3(vec3(warped + 3.0, t + 2.0));
  // slow channel with its own warp — the zebra source
  col.a = simplex3(vec3(p + wx, uTime * 0.005 + 3.0));

  gl_FragColor = col * 0.5 + 0.5;
}
`;

export function createFlowNoise(renderer) {
  return new ProceduralTexture(renderer, {
    width: 64,
    height: 64,
    fragmentShader: frag,
    type: THREE.HalfFloatType,
    wrap: THREE.RepeatWrapping
  });
}
