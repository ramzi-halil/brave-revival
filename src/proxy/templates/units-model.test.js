import { expect, mock, test } from 'bun:test';

mock.module('three', () => ({}));
mock.module('three/addons/controls/OrbitControls.js', () => ({ OrbitControls: class {} }));
mock.module('three/addons/effects/OutlineEffect.js', () => ({ OutlineEffect: class {} }));
const { resolveSkinBones } = await import('./units-model.js');

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
