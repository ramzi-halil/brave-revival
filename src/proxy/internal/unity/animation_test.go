package unity

import (
	"encoding/json"
	"math"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAnimationClip(t *testing.T) {
	data, err := os.ReadFile("testdata/animation.unity3d")
	require.NoError(t, err)
	for _, test := range []struct {
		container, clip string
		position        []float32
	}{
		{"first", "101", []float32{0, 1, 2, -.5, 1, 2, -1, 1, 2, -1.5, 1, 2, -2, 1, 2}},
		{"second", "101", []float32{0, 2, 3, -.15625, 2, 3, -.5, 2, 3, -.84375, 2, 3, -1, 2, 3}},
		{"direct", "sampled", []float32{0, 0, 0, -.5, 1, 1.5, -1, 2, 3, -1.5, 3, 4.5, -2, 4, 6}},
	} {
		t.Run(test.container, func(t *testing.T) {
			clip, err := AnimationClip(data, "test", test.container, test.clip)
			require.NoError(t, err)
			require.Equal(t, test.clip, clip.Name)
			require.Regexp(t, `^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`, clip.UUID.String())
			encoded, err := json.Marshal(clip)
			require.NoError(t, err)
			require.Contains(t, string(encoded), `"uuid":"`+clip.UUID.String()+`"`)
			require.Equal(t, 1.0, clip.Duration)
			require.Equal(t, 2500, clip.BlendMode)
			require.Len(t, clip.Tracks, 3)
			require.Equal(t, AnimationTrack{"1509296682.position", "vector", []float32{0, .25, .5, .75, 1}, test.position}, clip.Tracks[0])
			require.Equal(t, "1509296682.quaternion", clip.Tracks[1].Name)
			require.Equal(t, "quaternion", clip.Tracks[1].Type)
			require.Equal(t, []float32{0, 1}, clip.Tracks[1].Times)
			if test.container == "first" {
				require.InDeltaSlice(t, []float32{0, -1.0 / 3, -2.0 / 3, 2.0 / 3, 0, -1.0 / 3, -2.0 / 3, 2.0 / 3}, clip.Tracks[1].Values, 1e-7)
			} else {
				require.Equal(t, []float32{0, 0, 0, 1, 0, 0, 0, 1}, clip.Tracks[1].Values)
			}
			require.Equal(t, AnimationTrack{"1509296682.scale", "vector", []float32{0, 1}, []float32{1, 1, 1, 1, 1, 1}}, clip.Tracks[2])
		})
	}
	for container, name := range map[string]string{"first": "101", "second": "101", "direct": "sampled", "5": "101"} {
		clip, err := AnimationClip(data, "test", container, "")
		require.NoError(t, err)
		require.Equal(t, name, clip.Name)
	}
	clip, err := AnimationClip(data, "test", "assets/test", "")
	require.NoError(t, err)
	require.Contains(t, []string{"101", "sampled"}, clip.Name)
	for _, test := range []struct{ container, clip string }{{"absent", "101"}, {"absent", ""}, {"first", "sampled"}, {"first", "10"}} {
		_, err := AnimationClip(data, "test", test.container, test.clip)
		require.ErrorIs(t, err, ErrAnimationNotFound)
	}
	_, err = AnimationClip(data, "test", "assets/test", "101")
	require.EqualError(t, err, `ambiguous AnimationClip "101" in container "assets/test"`)
	for _, name := range []string{"101", ""} {
		clip, err := AnimationClip(data, "test", "timeline/character", name)
		require.NoError(t, err)
		require.Equal(t, []float32{0, 1, 2, -.5, 1, 2, -1, 1, 2, -1.5, 1, 2, -2, 1, 2}, clip.Tracks[0].Values)
	}
	for _, container := range []string{"timeline/ambiguous", "timeline/non-timeline"} {
		_, err := AnimationClip(data, "test", container, "101")
		require.ErrorContains(t, err, "ambiguous AnimationClip")
	}
	for _, broken := range [][]byte{nil, []byte("invalid"), []byte("UnityFS\x00"), data[:len(data)-1]} {
		_, err = AnimationClip(broken, "broken", "first", "101")
		require.Error(t, err)
	}
}

func TestAnimationCurveValidation(t *testing.T) {
	data, err := os.ReadFile("testdata/animation.unity3d")
	require.NoError(t, err)
	files, err := readBundle(data)
	require.NoError(t, err)
	bits := func(value float32) int64 { return int64(math.Float32bits(value)) }
	for _, test := range []struct {
		name, failure string
		mutate        func(map[string]any)
	}{
		{"time range", "invalid animation time range", func(v map[string]any) {
			v["m_MuscleClip"].(map[string]any)["m_StopTime"] = bits(1)
		}},
		{"sample limit", "invalid animation time range", func(v map[string]any) { v["m_SampleRate"] = bits(1e20) }},
		{"float curve", "unsupported animation curves: m_FloatCurves", func(v map[string]any) { v["m_FloatCurves"] = []any{nil} }},
		{"weighted keys", "unsupported curve keys", func(v map[string]any) {
			curve := v["m_PositionCurves"].([]any)[0].(map[string]any)["curve"].(map[string]any)
			curve["m_Curve"].([]any)[0].(map[string]any)["weightedMode"] = int64(1)
		}},
		{"duplicate times", "unsupported curve keys", func(v map[string]any) {
			curve := v["m_PositionCurves"].([]any)[0].(map[string]any)["curve"].(map[string]any)
			curve["m_Curve"].([]any)[1].(map[string]any)["time"] = bits(1)
		}},
	} {
		t.Run(test.name, func(t *testing.T) {
			err := readAssetObjects(files[0].data, func(typ assetType, object *reader, _ unityVersion, _ int, id int64) bool {
				if id != 1 {
					return false
				}
				values := readTreeValue(object, typ.nodes, 0).(map[string]any)
				test.mutate(values)
				require.PanicsWithError(t, test.failure, func() { animationFromValues(values) })
				return true
			})
			require.NoError(t, err)
		})
	}
}

func TestSampleAnimation(t *testing.T) {
	keys := []animationKey{{time: 1, value: 2, outSlope: 0}, {time: 2, value: 4, inSlope: 0}}
	require.Equal(t, 2.3125, sampleAnimation(keys, 1.25, false))
	require.Equal(t, 2.5, sampleAnimation(keys, 1.25, true))
	require.Equal(t, 2.0, sampleAnimation(keys, 0, false))
	require.Equal(t, 4.0, sampleAnimation(keys, 3, false))
	keys[1].inSlope = math.Inf(1)
	require.Equal(t, 2.0, sampleAnimation(keys, 1.25, false))
	require.Equal(t, 4.0, sampleAnimation(keys, 2, false))
}

func TestAnimationTargets(t *testing.T) {
	require.Equal(t, "", animationTarget(0))
	require.Equal(t, "385153371", animationTarget(385153371))
	require.Equal(t, "1509296682", animationTarget(1509296682))
	require.Equal(t, "face_UV", animationTarget(1189889199))
}

func TestAnimationQuaternionAndStreamedTangents(t *testing.T) {
	data, err := os.ReadFile("testdata/animation.unity3d")
	require.NoError(t, err)
	files, err := readBundle(data)
	require.NoError(t, err)
	bits := func(value float32) int64 { return int64(math.Float32bits(value)) }
	err = readAssetObjects(files[0].data, func(typ assetType, object *reader, _ unityVersion, _ int, id int64) bool {
		if id != 1 && id != 2 {
			return false
		}
		values := readTreeValue(object, typ.nodes, 0).(map[string]any)
		if id == 1 {
			curve := values["m_RotationCurves"].([]any)[0].(map[string]any)["curve"].(map[string]any)
			for i, raw := range curve["m_Curve"].([]any) {
				w := float32(1)
				if i == 1 {
					w = -1
				}
				raw.(map[string]any)["value"] = map[string]any{"x": bits(0), "y": bits(0), "z": bits(0), "w": bits(w)}
			}
			clip := animationFromValues(values)
			require.Equal(t, []float32{0, 1}, clip.Tracks[1].Times)
			require.Equal(t, []float32{0, 0, 0, 1, 0, 0, 0, 1}, clip.Tracks[1].Values)
		} else {
			packed := values["m_MuscleClip"].(map[string]any)["m_Clip"].(map[string]any)["data"].(map[string]any)
			words := packed["m_StreamedClip"].(map[string]any)["data"].([]any)
			words[20], words[21] = bits(1), bits(0)
			clip := animationFromValues(values)
			require.Equal(t, []float32{0, 2, 3, -.015625, 2, 3, -.125, 2, 3, -.421875, 2, 3, -1, 2, 3}, clip.Tracks[0].Values)
		}
		return false
	})
	require.NoError(t, err)
}

func FuzzAnimationClip(f *testing.F) {
	for _, name := range []string{"animation.unity3d", "model.unity3d", "tree-be.assets"} {
		data, err := os.ReadFile("testdata/" + name)
		if err != nil {
			f.Fatal(err)
		}
		f.Add(data)
	}
	f.Fuzz(func(t *testing.T, data []byte) {
		AnimationClip(data, "test", "first", "101")
	})
}
