import * as THREE from 'three';
import { OrbitControls } from 'three/addons/controls/OrbitControls.js';
import { OutlineEffect } from 'three/addons/effects/OutlineEffect.js';

function dispose(group) {
    const materials = new Set();
    group.traverse(object => {
        if (!object.isMesh) return;
        object.geometry.dispose();
        materials.add(object.material);
    });
    for (const material of materials) {
        material.map?.dispose();
        material.dispose();
    }
}

async function loadPart(part, signal) {
    const response = await fetch(`/res/m/${part.id}`, { signal });
    if (!response.ok) throw new Error(await response.text());
    const data = await response.json();
    signal.throwIfAborted();
    const texture = await new THREE.TextureLoader().loadAsync(data.texture);
    texture.colorSpace = THREE.SRGBColorSpace;
    texture.wrapS = texture.wrapT = THREE.RepeatWrapping;
    const material = new THREE.MeshBasicMaterial({ map: texture, side: THREE.DoubleSide });
    const group = new THREE.Group();
    group.name = part.name;
    group.userData.resourceId = part.id;
    try {
        for (const mesh of data.meshes) {
            const geometry = new THREE.BufferGeometryLoader().parse(mesh.geometry);
            geometry.getAttribute('uv').setUsage(THREE.DynamicDrawUsage);
            if (!geometry.hasAttribute('normal')) geometry.computeVertexNormals();
            const object = new THREE.Mesh(geometry, material);
            object.name = mesh.name;
            group.add(object);
        }
        signal.throwIfAborted();
        return group;
    } catch (error) {
        dispose(group);
        if (!group.children.length) {
            texture.dispose();
            material.dispose();
        }
        throw error;
    }
}

export class UnitModel {
    constructor(container) {
        this.container = container;
        this.scene = new THREE.Scene();
        this.scene.background = new THREE.Color(0xf2f3f5);
        this.camera = new THREE.PerspectiveCamera(35, 1, 0.01, 1000);
        this.renderer = new THREE.WebGLRenderer({ antialias: true });
        this.renderer.setPixelRatio(Math.min(window.devicePixelRatio, 2));
        this.outline = new OutlineEffect(this.renderer, { defaultThickness: 0.003, defaultColor: [0, 0, 0] });
        this.renderer.domElement.setAttribute('aria-label', 'Unit figurine; drag to rotate, scroll to zoom');
        container.append(this.renderer.domElement);
        this.controls = new OrbitControls(this.camera, this.renderer.domElement);
        this.controls.enableDamping = true;
        this.scene.add(new THREE.HemisphereLight(0xffffff, 0x777788, 2));
        const light = new THREE.DirectionalLight(0xffffff, 2);
        light.position.set(-10, 20, 20);
        this.scene.add(light);
        this.group = new THREE.Group();
        this.scene.add(this.group);
        this.observer = new ResizeObserver(() => this.resize());
        this.observer.observe(container);
        this.resize();
        this.renderer.setAnimationLoop(() => {
            this.controls.update();
            this.outline.render(this.scene, this.camera);
        });
    }

    resize() {
        const width = this.container.clientWidth;
        const height = this.container.clientHeight;
        if (!width || !height) return;
        this.renderer.setSize(width, height, false);
        this.camera.aspect = width / height;
        this.camera.updateProjectionMatrix();
    }

    async show(parts, unitId) {
        this.controller?.abort();
        const controller = new AbortController();
        this.controller = controller;
        dispose(this.group);
        this.group.clear();
        const results = await Promise.allSettled(parts.map(part => loadPart(part, controller.signal)));
        if (controller.signal.aborted) {
            for (const result of results) if (result.status === 'fulfilled') dispose(result.value);
            return;
        }
        const failed = [];
        results.forEach((result, index) => {
            if (result.status === 'fulfilled') this.group.add(result.value);
            else failed.push(`${parts[index].name}: ${result.reason.message || 'Texture failed to load'}`);
        });
        if (this.group.children.length && this.unitId !== unitId) {
            const bounds = new THREE.Box3().setFromObject(this.group);
            const center = bounds.getCenter(new THREE.Vector3());
            const size = bounds.getSize(new THREE.Vector3());
            const verticalFov = THREE.MathUtils.degToRad(this.camera.fov);
            const horizontalFov = 2 * Math.atan(Math.tan(verticalFov / 2) * this.camera.aspect);
            const distance = 1.2 * Math.max(size.y / (2 * Math.tan(verticalFov / 2)), size.x / (2 * Math.tan(horizontalFov / 2))) + size.z / 2;
            this.camera.near = Math.max(distance / 1000, 0.001);
            this.camera.far = Math.max(distance * 100, 100);
            this.camera.position.copy(center).add(new THREE.Vector3(0, 0, distance));
            this.camera.updateProjectionMatrix();
            this.controls.target.copy(center);
            this.controls.minDistance = distance / 10;
            this.controls.maxDistance = distance * 10;
            this.controls.update();
            this.unitId = unitId;
        }
        return failed.length ? `Could not load ${failed.join('; ')}` : '';
    }
}
