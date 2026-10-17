import * as THREE from 'three';
import { OrbitControls } from 'three/addons/controls/OrbitControls.js';
import { OutlineEffect } from 'three/addons/effects/OutlineEffect.js';

function createSkybox() {
    const faces = Array.from({ length: 6 }, (_, index) => {
        const canvas = document.createElement('canvas');
        canvas.width = canvas.height = 512;
        const context = canvas.getContext('2d');
        const gradient = context.createLinearGradient(0, 0, 0, 512);
        gradient.addColorStop(0, '#adcbe1');
        gradient.addColorStop(0.48, '#edf4f8');
        gradient.addColorStop(0.5, '#bac8d2');
        gradient.addColorStop(0.52, '#dce0e3');
        gradient.addColorStop(1, '#929fa9');
        context.fillStyle = index === 2 ? '#adcbe1' : index === 3 ? '#929fa9' : gradient;
        context.fillRect(0, 0, 512, 512);
        context.strokeStyle = '#64788c30';
        context.lineWidth = 1;
        context.beginPath();
        context.moveTo(256, 0);
        context.lineTo(256, 512);
        context.moveTo(0, 256);
        context.lineTo(512, 256);
        context.stroke();
        return canvas;
    });
    const texture = new THREE.CubeTexture(faces);
    texture.colorSpace = THREE.SRGBColorSpace;
    texture.needsUpdate = true;
    return texture;
}

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
    const restPose = new Map();
    root.traverse(node => {
        for (const property of ['position', 'quaternion', 'scale']) {
            const value = node[property].toArray();
            restPose.set(`${node.uuid}.${property}`, value);
            if (node === root || node.name) restPose.set(`${node === root ? '' : node.name}.${property}`, value);
        }
    });
    return { root, bones, restInverses, faceUV, restPose };
}

export function stitchAnimationClips(clips, restPose) {
    if (clips.length === 1) return clips[0];
    const segments = clips.map(clip => new Map(clip.tracks.map(track => [track.name, track])));
    const tracks = new Map(clips.flatMap(clip => clip.tracks.map(track => [track.name, track])));
    const combined = [];
    for (const [name, template] of tracks) {
        const size = template.getValueSize();
        const times = [], values = [];
        let offset = 0;
        for (let index = 0; index < clips.length; index++) {
            const track = segments[index].get(name);
            const duration = clips[index].duration;
            if (track) {
                if (track.constructor !== template.constructor || track.getInterpolation() !== template.getInterpolation()) {
                    throw new Error(`Incompatible animation track: ${name}`);
                }
                if (track.times[0] > 0) {
                    times.push(offset);
                    values.push(...track.values.slice(0, size));
                }
                for (const time of track.times) times.push(offset + Math.min(time, duration));
                for (const value of track.values) values.push(value);
                if (track.times.at(-1) < duration) {
                    times.push(offset + duration);
                    values.push(...track.values.slice(-size));
                }
            } else {
                const value = restPose.get(name) || template.values.slice(0, size);
                times.push(offset, offset + duration);
                values.push(...value, ...value);
            }
            offset += duration;
        }
        // Duplicate boundary times preserve each clip's endpoint without blending into the next pose.
        combined.push(new template.constructor(name, times, values, template.getInterpolation()));
    }
    return new THREE.AnimationClip(clips.map(clip => clip.name).join('+'), clips.reduce((sum, clip) => sum + clip.duration, 0), combined, clips[0].blendMode);
}

export class FaceUV {
    constructor(geometry) {
        this.uv = geometry.getAttribute('uv');
        this.eyes = [];
        this.mouth = [];
        for (let index = 0; index < this.uv.count; index++) {
            const u = this.uv.getX(index) - Math.floor(this.uv.getX(index));
            const v = this.uv.getY(index) - Math.floor(this.uv.getY(index));
            // Atlas coordinates may be outside [0, 1] because the texture repeats.
            if (u < 0.72 && v > 0.75) this.eyes.push([index, this.uv.getY(index)]);
            else if (u >= 0.72 && u < 0.89 && v > 0.9) this.mouth.push([index, this.uv.getY(index)]);
        }
    }

    update(position) {
        const eye = (-position.x) & 3;
        const mouth = position.y & 3;
        if (eye === this.eyeIndex && mouth === this.mouthIndex) return;
        for (const [index, v] of this.eyes) this.uv.setY(index, v - eye * 0.25);
        for (const [index, v] of this.mouth) this.uv.setY(index, v - mouth * 0.10);
        this.eyeIndex = eye;
        this.mouthIndex = mouth;
        this.uv.needsUpdate = true;
    }
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
            if (mesh.name === 'f') object.userData.faceUV = new FaceUV(geometry);
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
        this.scene.background = createSkybox();
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
        const checker = new THREE.DataTexture(new Uint8Array([
            220, 224, 227, 255, 186, 200, 210, 255,
            186, 200, 210, 255, 220, 224, 227, 255,
        ]), 2, 2);
        checker.colorSpace = THREE.SRGBColorSpace;
        checker.wrapS = checker.wrapT = THREE.RepeatWrapping;
        checker.magFilter = THREE.NearestFilter;
        checker.minFilter = THREE.LinearMipmapLinearFilter;
        checker.generateMipmaps = true;
        checker.needsUpdate = true;
        const floorMaterial = new THREE.MeshBasicMaterial({ map: checker, side: THREE.FrontSide });
        floorMaterial.userData.outlineParameters = { visible: false };
        this.floor = new THREE.Mesh(new THREE.PlaneGeometry(1, 1), floorMaterial);
        this.floor.rotation.x = -Math.PI / 2;
        this.floor.visible = false;
        this.scene.add(this.floor);
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
            this.updateFaceUV();
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
        let clips;
        if (path) {
            const sources = path.split('+').map(source => source.trim().split('/'));
            if (sources.some(parts => parts.length > 3 || !/^\d+$/.test(parts[0]) || parts.some(part => !part || part === '.' || part === '..'))) {
                throw new Error('Enter id, id/container, or id/container/clip');
            }
            clips = await Promise.all(sources.map(async parts => {
                const response = await fetch(`/res/a/${parts.map(encodeURIComponent).join('/')}`, { signal: controller.signal });
                if (!response.ok) throw new Error(await response.text());
                const clip = THREE.AnimationClip.parse(await response.json());
                clip.tracks = clip.tracks.filter(track => !track.name.startsWith('face_UV.') || track.name === 'face_UV.position');
                if (!Number.isFinite(clip.duration) || clip.duration <= 0 || !clip.validate()) throw new Error('Invalid animation clip');
                return clip;
            }));
        }
        const armature = await this.getArmature();
        controller.signal.throwIfAborted();
        if (clips) clip = stitchAnimationClips(clips, armature.restPose);
        this.mixer.stopAllAction();
        if (this.animationClip) this.mixer.uncacheClip(this.animationClip);
        this.animationClip = clip;
        this.animationAction = clip ? this.mixer.clipAction(clip).reset().play() : undefined;
        this.animationPath = path;
        this.playing = !!clip;
        this.mixer.update(0);
        this.fitCamera();
        this.updateFaceUV();
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
        this.updateFaceUV();
        this.onAnimationChange?.(this.animationState);
    }

    updateFaceUV() {
        if (!this.armature) return;
        this.group.traverse(object => object.userData.faceUV?.update(this.armature.faceUV.position));
    }

    resize() {
        const width = this.container.clientWidth;
        const height = this.container.clientHeight;
        if (!width || !height) return;
        this.renderer.setSize(width, height, false);
        this.camera.aspect = width / height;
        this.camera.updateProjectionMatrix();
    }

    modelBounds() {
        const bounds = new THREE.Box3();
        const action = this.animationAction;
        const time = action?.time;
        const times = new Set([0]);
        if (action) {
            const duration = this.animationClip.duration;
            const frames = Math.ceil(duration * 60);
            for (let frame = 1; frame <= frames; frame++) times.add(duration * frame / frames);
            for (const track of this.animationClip.tracks) {
                for (const keyTime of track.times) times.add(keyTime);
            }
        }
        try {
            for (const sample of [...times].sort((a, b) => a - b)) {
                if (action) {
                    action.time = sample;
                    this.mixer.update(0);
                }
                this.scene.updateMatrixWorld(true);
                // Precise bounds evaluate skinning instead of reusing a previous pose's cached box.
                bounds.expandByObject(this.group, true);
            }
        } finally {
            if (action) {
                action.time = time;
                this.mixer.update(0);
                this.scene.updateMatrixWorld(true);
            }
        }
        return bounds;
    }

    updateFloor(bounds) {
        this.floor.visible = !bounds.isEmpty();
        if (!this.floor.visible) return;
        const minX = Math.min(bounds.min.x, -5);
        const minZ = Math.min(bounds.min.z, -5);
        const maxX = Math.max(bounds.max.x, 5);
        const maxZ = Math.max(bounds.max.z, 5);
        const width = maxX - minX;
        const depth = maxZ - minZ;
        this.floor.position.set((minX + maxX) / 2, 0, (minZ + maxZ) / 2);
        this.floor.scale.set(width, depth, 1);
        this.floor.material.map.repeat.set(width / 10, depth / 10);
        this.floor.material.map.offset.set(-width / 20, -depth / 20);
    }

    fitCamera(resetAngle = false) {
        const bounds = this.modelBounds();
        this.updateFloor(bounds);
        if (bounds.isEmpty()) return;
        const center = bounds.getCenter(new THREE.Vector3());
        if (resetAngle) this.camera.quaternion.identity();
        const rotation = new THREE.Matrix4().makeRotationFromQuaternion(this.camera.quaternion.clone().invert());
        const size = bounds.clone().applyMatrix4(rotation).getSize(new THREE.Vector3());
        const verticalTan = Math.tan(THREE.MathUtils.degToRad(this.camera.fov) / 2);
        const horizontalTan = verticalTan * this.camera.aspect;
        const distance = 1.2 * Math.max(size.y / (2 * verticalTan), size.x / (2 * horizontalTan)) + size.z / 2;
        this.camera.near = Math.max(distance / 1000, 0.001);
        this.camera.far = Math.max(distance * 100, 100);
        this.camera.position.copy(center).add(new THREE.Vector3(0, 0, distance).applyQuaternion(this.camera.quaternion));
        this.camera.updateProjectionMatrix();
        this.controls.target.copy(center);
        this.controls.minDistance = distance / 10;
        this.controls.maxDistance = distance * 10;
        this.controls.update();
    }

    async show(parts, unitId) {
        this.controller?.abort();
        const controller = new AbortController();
        this.controller = controller;
        dispose(this.group);
        this.group.clear();
        this.floor.visible = false;
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
            this.fitCamera(true);
            this.unitId = unitId;
        } else {
            this.updateFloor(this.modelBounds());
        }
        this.updateFaceUV();
        return failed.length ? `Could not load ${failed.join('; ')}` : '';
    }
}
