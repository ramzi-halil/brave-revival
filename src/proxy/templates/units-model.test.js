import { expect, mock, test } from 'bun:test';

mock.module('three', () => ({}));
mock.module('three/addons/controls/OrbitControls.js', () => ({ OrbitControls: class {} }));
mock.module('three/addons/effects/OutlineEffect.js', () => ({ OutlineEffect: class {} }));
const { FaceUV, resolveSkinBones } = await import('./units-model.js');

function faceGeometry() {
    const array = new Float64Array([0.1, -0.05, 1.7, 1.9, -0.2, 0.95, 0.8, 0.4, 0.94, 0.95, 0.75, 0.1]);
    const uv = {
        array, count: array.length / 2,
        getX: index => array[index * 2],
        getY: index => array[index * 2 + 1],
        setY: (index, value) => { array[index * 2 + 1] = value; },
    };
    return { getAttribute: () => uv };
}

test('face expressions shift only eye and mouth atlas regions, without accumulating offsets', () => {
    const geometry = faceGeometry();
    const uv = geometry.getAttribute('uv');
    const original = uv.array.slice();
    const face = new FaceUV(geometry);
    face.update({ x: -2, y: 3 });
    expect(uv.getY(0)).toBeCloseTo(-0.55);
    expect(uv.getY(1)).toBeCloseTo(1.4);
    expect(uv.getY(2)).toBeCloseTo(0.65);
    for (const index of [3, 4, 5]) expect(uv.getY(index)).toBe(original[index * 2 + 1]);
    for (let index = 0; index < uv.count; index++) expect(uv.getX(index)).toBe(original[index * 2]);
    expect(uv.needsUpdate).toBe(true);

    face.update({ x: -1, y: 1 });
    expect(uv.getY(0)).toBeCloseTo(-0.3);
    expect(uv.getY(2)).toBeCloseTo(0.85);
    face.update({ x: 0, y: 0 });
    expect(uv.array).toEqual(original);
});

test('face selectors stay integral and within 0–3 during interpolation', () => {
    const geometry = faceGeometry();
    const uv = geometry.getAttribute('uv');
    const face = new FaceUV(geometry);
    face.update({ x: -0.51, y: 1.49 });
    expect(uv.getY(0)).toBeCloseTo(-0.3);
    expect(uv.getY(2)).toBeCloseTo(0.85);
    face.update({ x: 1, y: 4 });
    expect(uv.getY(0)).toBe(-0.05);
    expect(uv.getY(2)).toBeCloseTo(0.65);
    uv.needsUpdate = false;
    face.update({ x: 2, y: 5 });
    expect(uv.needsUpdate).toBe(false);
});

const translation = y => [1, 0, 0, 0, 0, 1, 0, 0, 0, 0, 1, 0, 0, y, 0, 1];
const root = { name: 'root' }, hip = { name: 'hip' }, head = { name: 'head' };
const armature = {
    root,
    bones: new Map([[1, root], [2, hip], [3, head]]),
    restInverses: new Map([[1, translation(0)], [2, translation(-7)], [3, translation(-12)]]),
};
const geometry = (indices, weights) => ({
    getAttribute: name => ({ array: name === 'skinIndex' ? indices : weights }),
});

test('repairs a rotated hash palette only when every inverse bind matches the rig', () => {
    const mesh = { boneNameHashes: [2, 3, 1], bindPoses: [translation(0), translation(-7), translation(-12)] };
    expect(resolveSkinBones(mesh, geometry([2, 1, 0, 0], [0.8, 0.2, 0, 0]), armature)).toEqual([root, hip, head]);
    expect(mesh.boneNameHashes).toEqual([2, 3, 1]);

    mesh.bindPoses[2] = translation(-13);
    expect(resolveSkinBones(mesh, geometry([2], [1]), armature)).toEqual([hip, head, root]);
});

test('retains an authored bind pose that differs from the base rig', () => {
    const mesh = { boneNameHashes: [1, 2, 3], bindPoses: [translation(0), translation(-8), translation(-12)] };
    expect(resolveSkinBones(mesh, geometry([1], [1]), armature)).toEqual([root, hip, head]);
    expect(mesh.bindPoses[1]).toEqual(translation(-8));
});

test('tolerates missing attachment bones only when their palette slots have zero weight', () => {
    const mesh = { boneNameHashes: [1, 986342626, 3], bindPoses: [translation(0), translation(-7), translation(-12)] };
    expect(resolveSkinBones(mesh, geometry([2, 1, 0, 0], [1, 0, 0, 0]), armature)).toEqual([root, root, head]);
    expect(() => resolveSkinBones(mesh, geometry([2, 1], [0.9, 0.1]), armature))
        .toThrow('Skeleton is missing bone path hash 986342626');
});
