import * as THREE from 'three';
import { GLSL_QUAD_VERT } from './shader-lib.js';

/**
 * A small render-target texture driven by a fragment shader, re-renderable
 * every frame. Used for the shared flow-noise texture and the wall patterns.
 */

const quadGeo = new THREE.BufferGeometry();
quadGeo.setAttribute('position', new THREE.BufferAttribute(new Float32Array([
  -1, -1, 0, 3, -1, 0, -1, 3, 0
]), 3));
quadGeo.setAttribute('uv', new THREE.BufferAttribute(new Float32Array([
  0, 0, 2, 0, 0, 2
]), 2));

export class ProceduralTexture {
  constructor(renderer, {
    width = 64,
    height = 64,
    fragmentShader,
    uniforms = {},
    type = THREE.HalfFloatType,
    wrap = THREE.RepeatWrapping
  }) {
    this.renderer = renderer;
    this.rt = new THREE.WebGLRenderTarget(width, height, {
      type,
      wrapS: wrap,
      wrapT: wrap,
      minFilter: THREE.LinearFilter,
      magFilter: THREE.LinearFilter,
      depthBuffer: false
    });
    this.uniforms = {
      uTime: { value: 0 },
      uScreenAspectRatio: { value: 1 },
      ...uniforms
    };
    this.scene = new THREE.Scene();
    this.camera = new THREE.OrthographicCamera(-1, 1, 1, -1, 0, 1);
    this.mesh = new THREE.Mesh(quadGeo, new THREE.RawShaderMaterial({
      // RawShaderMaterial would need attribute decls; use ShaderMaterial instead.
    }));
    this.mesh.material.dispose();
    this.mesh.material = new THREE.ShaderMaterial({
      vertexShader: GLSL_QUAD_VERT,
      fragmentShader,
      uniforms: this.uniforms,
      depthTest: false,
      depthWrite: false
    });
    this.mesh.frustumCulled = false;
    this.scene.add(this.mesh);
  }

  get texture() {
    return this.rt.texture;
  }

  setSize(w, h) {
    this.rt.setSize(Math.max(1, Math.round(w)), Math.max(1, Math.round(h)));
  }

  render(time) {
    if (time !== undefined) this.uniforms.uTime.value = time;
    const prev = this.renderer.getRenderTarget();
    this.renderer.setRenderTarget(this.rt);
    this.renderer.render(this.scene, this.camera);
    this.renderer.setRenderTarget(prev);
  }

  dispose() {
    this.rt.dispose();
    this.mesh.material.dispose();
  }
}
