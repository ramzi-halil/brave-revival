import * as THREE from 'three';
import { OrbitControls } from 'three/addons/controls/OrbitControls.js';
import { OutlineEffect } from 'three/addons/effects/OutlineEffect.js';

function dispose(group) {
    const materials = new Set();
    group.traverse(object => {
        if (!object.isMesh) return;
        object.geometry.dispose();
        if (object.isSkinnedMesh) object.skeleton?.dispose();
        materials.add(object.material);
    });
    for (const material of materials) {
        material.map?.dispose();
        material.dispose();
    }
}

async function loadArmature() {
    const response = await fetch('/skeleton.json');
    if (!response.ok) throw new Error(await response.text());
    const root = new THREE.ObjectLoader().parse(await response.json());
    const bones = new Map();
    root.traverse(bone => {
        if (bone.isBone) bones.set(Number(bone.uuid), bone);
    });
    root.updateMatrixWorld(true);
    const restInverses = new Map([...bones].map(([hash, bone]) => [hash, bone.matrixWorld.clone().invert().elements]));
    const faceUV = new THREE.Object3D();
    faceUV.name = 'face_UV';
    root.add(faceUV);
    return { root, bones, restInverses };
}

export function resolveSkinBones(mesh, geometry, armature) {
    let hashes = mesh.boneNameHashes;
    const matchesBindPose = (hash, index) => armature.restInverses.get(hash)?.every((value, component) =>
        Math.abs(value - mesh.bindPoses[index][component]) < 0.0001);
    // Some bundles move the root hash to the end without moving weights or bind poses.
    if (!hashes.every(matchesBindPose)) {
        for (let offset = 1; offset < hashes.length; offset++) {
            const rotated = hashes.slice(offset).concat(hashes.slice(0, offset));
            if (rotated.every(matchesBindPose)) {
                hashes = rotated;
                break;
            }
        }
    }
    const indices = geometry.getAttribute('skinIndex').array;
    const weights = geometry.getAttribute('skinWeight').array;
    const used = new Set();
    for (let i = 0; i < weights.length; i++) if (weights[i] > 0) used.add(indices[i]);
    return hashes.map((hash, index) => {
        const bone = armature.bones.get(hash);
        if (!bone && used.has(index)) throw new Error(`Skeleton is missing bone path hash ${hash}`);
        // Zero-weight palette slots may name attachment bones absent from the base rig.
        return bone || armature.root;
    });
}

async function loadPart(part, signal, armature) {
    const query = part.assetNames.map(n => 'name=' + encodeURIComponent(n)).join('&');
    const response = await fetch(`/res/m/${part.id}?${query}`, { signal });
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
            const object = mesh.bindPoses ? new THREE.SkinnedMesh(geometry, material) : new THREE.Mesh(geometry, material);
            object.name = mesh.name;
            group.add(object);
            if (object.isSkinnedMesh) {
                object.frustumCulled = false;
                const bones = resolveSkinBones(mesh, geometry, armature);
                const inverses = mesh.bindPoses.map(matrix => new THREE.Matrix4().fromArray(matrix));
                // Parts share bone transforms, but retain their own Unity palette and inverse binds.
                object.bindMode = THREE.DetachedBindMode;
                object.bind(new THREE.Skeleton(bones, inverses), new THREE.Matrix4());
                object.normalizeSkinWeights();
            }
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
        this.playing = false;
        this.renderer.setAnimationLoop(time => {
            const delta = Math.min((time - (this.frameTime ?? time)) / 1000, 0.1);
            this.frameTime = time;
            if (this.playing && this.animationAction) {
                this.mixer.update(delta);
                this.onAnimationChange?.(this.animationState);
            }
            this.controls.update();
            this.outline.render(this.scene, this.camera);
        });
    }

    async getArmature() {
        const armature = await (this.armaturePromise ||= loadArmature());
        if (!this.mixer) {
            this.scene.add(armature.root);
            this.armature = armature;
            this.mixer = new THREE.AnimationMixer(armature.root);
        }
        return armature;
    }

    get animationState() {
        return {
            duration: this.animationClip?.duration || 0,
            time: this.animationAction?.time || 0,
            playing: this.playing,
        };
    }

    async loadAnimation(path) {
        this.animationController?.abort();
        const controller = new AbortController();
        this.animationController = controller;
        path = path.trim();
        let clip;
        if (path) {
            const parts = path.split('/');
            if (parts.length > 3 || !/^\d+$/.test(parts[0]) || parts.some(part => !part || part === '.' || part === '..')) {
                throw new Error('Enter id, id/container, or id/container/clip');
            }
            const response = await fetch(`/res/a/${parts.map(encodeURIComponent).join('/')}`, { signal: controller.signal });
            if (!response.ok) throw new Error(await response.text());
            clip = THREE.AnimationClip.parse(await response.json());
            if (!Number.isFinite(clip.duration) || clip.duration <= 0 || !clip.validate()) throw new Error('Invalid animation clip');
        }
        await this.getArmature();
        controller.signal.throwIfAborted();
        this.mixer.stopAllAction();
        if (this.animationClip) this.mixer.uncacheClip(this.animationClip);
        this.animationClip = clip;
        this.animationAction = clip ? this.mixer.clipAction(clip).reset().play() : undefined;
        this.animationPath = path;
        this.playing = !!clip;
        this.mixer.update(0);
        this.onAnimationChange?.(this.animationState);
    }

    toggleAnimation() {
        if (!this.animationAction) return;
        this.playing = !this.playing;
        this.onAnimationChange?.(this.animationState);
    }

    seekAnimation(time) {
        if (!this.animationAction || !Number.isFinite(time)) return;
        this.playing = false;
        this.animationAction.time = Math.max(0, Math.min(time, this.animationClip.duration));
        this.mixer.update(0);
        this.onAnimationChange?.(this.animationState);
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
        const armature = await this.getArmature();
        if (controller.signal.aborted) return;
        const results = await Promise.allSettled(parts.map(part => loadPart(part, controller.signal, armature)));
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
