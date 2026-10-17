package unity

import (
	"bytes"
	"errors"
	"fmt"
	"hash/crc32"
	"math"
	"slices"
	"sort"
	"strconv"
	"strings"
	"uuid"
)

var ErrAnimationNotFound = errors.New("animation clip not found")

type Animation struct {
	UUID      uuid.UUID        `json:"uuid"`
	Name      string           `json:"name"`
	Duration  float64          `json:"duration"`
	Tracks    []AnimationTrack `json:"tracks"`
	BlendMode int              `json:"blendMode"`
}

type AnimationTrack struct {
	Name          string    `json:"name"`
	Type          string    `json:"type"`
	Times         []float32 `json:"times"`
	Values        []float32 `json:"values"`
	Interpolation int       `json:"interpolation,omitempty"`
}

const interpolateDiscrete = 2300

func AnimationClip(data []byte, name, container, clip string) (animation *Animation, err error) {
	defer func() {
		if failure := recover(); failure != nil {
			animation = nil
			err = fmt.Errorf("failed to decode Unity animation: %v", failure)
		}
	}()
	files := []bundleFile{{name, data}}
	if bytes.HasPrefix(data, []byte("UnityFS\x00")) || bytes.HasPrefix(data, []byte("UnityRaw\x00")) || bytes.HasPrefix(data, []byte("UnityWeb\x00")) {
		files, err = readBundle(data)
		if err != nil {
			return nil, err
		}
	} else if !isSerialized(data) {
		return nil, fmt.Errorf("unsupported Unity asset file")
	}
	type candidate struct {
		values   map[string]any
		playable bool
	}
	var candidates []candidate
	for _, file := range files {
		if !isSerialized(file.data) {
			continue
		}
		clips := make(map[int64]map[string]any)
		refs := make(map[int64]bool)
		playableScripts := make(map[int64]bool)
		playables := make(map[int64]struct{ script, clip int64 })
		err = readAssetObjects(file.data, func(typ assetType, object *reader, _ unityVersion, _ int, pathID int64) bool {
			if typ.classID != 74 && typ.classID != 142 && typ.classID != 114 && typ.classID != 115 {
				return false
			}
			if len(typ.nodes) == 0 {
				if typ.classID == 114 || typ.classID == 115 {
					return false
				}
				panic(fmt.Errorf("Unity animation asset has no type tree"))
			}
			values := readTreeValue(object, typ.nodes, 0).(map[string]any)
			if typ.classID == 115 {
				playableScripts[pathID] = values["m_ClassName"] == "AnimationPlayableAsset" && values["m_Namespace"] == "UnityEngine.Timeline"
				return false
			}
			if typ.classID == 114 {
				script, hasScript := animationLocalRef(values["m_Script"])
				clip, hasClip := animationLocalRef(values["m_Clip"])
				if hasScript && hasClip {
					playables[pathID] = struct{ script, clip int64 }{script, clip}
				}
				return false
			}
			if typ.classID == 74 {
				if clip == "" || values["m_Name"] == clip {
					clips[pathID] = values
				}
				return false
			}
			addRef := func(raw any) {
				if id, ok := animationLocalRef(raw); ok {
					refs[id] = true
				}
			}
			preloads := values["m_PreloadTable"].([]any)
			for _, raw := range values["m_Container"].([]any) {
				entry := raw.(map[string]any)
				if !strings.Contains(entry["first"].(string), container) {
					continue
				}
				info := entry["second"].(map[string]any)
				start, size := int(info["preloadIndex"].(int64)), int(info["preloadSize"].(int64))
				if start < 0 || size < 0 || start > len(preloads) || size > len(preloads)-start {
					panic(fmt.Errorf("invalid Unity container preload range"))
				}
				addRef(info["asset"])
				for _, ref := range preloads[start : start+size] {
					addRef(ref)
				}
			}
			return false
		})
		if err != nil {
			return nil, err
		}
		preferred := make(map[int64]bool)
		for id, asset := range playables {
			if refs[id] && playableScripts[asset.script] {
				preferred[asset.clip] = true
			}
		}
		for id, values := range clips {
			if !refs[id] {
				continue
			}
			candidates = append(candidates, candidate{values, preferred[id]})
		}
	}
	// Preloads also include camera/effect dependencies; explicit Timeline animation assets take precedence.
	var selected map[string]any
	preferred, ambiguous := false, false
	for _, candidate := range candidates {
		if selected == nil || candidate.playable && !preferred {
			selected, preferred, ambiguous = candidate.values, candidate.playable, false
		} else if candidate.playable == preferred {
			ambiguous = true
		}
	}
	if selected == nil {
		return nil, fmt.Errorf("%w: %q in container %q", ErrAnimationNotFound, clip, container)
	}
	if clip != "" && ambiguous {
		return nil, fmt.Errorf("ambiguous AnimationClip %q in container %q", clip, container)
	}
	return animationFromValues(selected), nil
}

func animationLocalRef(raw any) (int64, bool) {
	ref, ok := raw.(map[string]any)
	if !ok || ref["m_FileID"] != int64(0) {
		return 0, false
	}
	id, ok := ref["m_PathID"].(int64)
	return id, ok && id != 0
}

type animationKey struct {
	time, value, inSlope, outSlope float64
	stepped                        bool
}

type animationCurve struct {
	target     string
	property   int
	components [][]animationKey
	linear     bool
}

func animationFloat(raw any) float64 {
	return float64(math.Float32frombits(uint32(raw.(int64))))
}

func animationFinite(value float64) bool { return !math.IsNaN(value) && !math.IsInf(value, 0) }

func animationTarget(hash uint32) string {
	switch hash {
	case 0:
		return ""
	case crc32.ChecksumIEEE([]byte("face_UV")):
		// This texture-atlas control is not part of the armature.
		return "face_UV"
	default:
		return strconv.FormatUint(uint64(hash), 10)
	}
}

func animationFromValues(values map[string]any) *Animation {
	for _, name := range []string{"m_CompressedRotationCurves", "m_EulerCurves", "m_FloatCurves", "m_PPtrCurves"} {
		if curves, _ := values[name].([]any); len(curves) != 0 {
			panic(fmt.Errorf("unsupported animation curves: %s", name))
		}
	}
	muscle := values["m_MuscleClip"].(map[string]any)
	start, stop := animationFloat(muscle["m_StartTime"]), animationFloat(muscle["m_StopTime"])
	rate := animationFloat(values["m_SampleRate"])
	if !animationFinite(start) || !animationFinite(stop) || !animationFinite(rate) || rate <= 0 || stop <= start ||
		!animationFinite(float64(float32(stop-start))) || (stop-start)*rate > maxAssetSize/16 {
		panic(fmt.Errorf("invalid animation time range"))
	}
	packed := muscle["m_Clip"].(map[string]any)
	if data, ok := packed["data"].(map[string]any); ok {
		packed = data
	}
	dense, _ := packed["m_DenseClip"].(map[string]any)
	baked := dense != nil && dense["m_CurveCount"].(int64) > 0
	curves := explicitAnimationCurves(values)
	if len(curves) == 0 {
		curves = muscleAnimationCurves(packed, values["m_ClipBindingConstant"].(map[string]any), start, stop)
	}
	if len(curves) == 0 {
		panic(fmt.Errorf("no transform curves found"))
	}
	animation := &Animation{
		UUID: uuid.New(),
		Name: values["m_Name"].(string), Duration: stop - start, BlendMode: 2500,
	}
	budget := maxAssetSize / 4
	for _, curve := range curves {
		discrete := curve.target == "face_UV"
		times := []float64{start, stop}
		for frame := 0; !discrete && start+float64(frame)/rate < stop; frame++ {
			times = append(times, start+float64(frame)/rate)
		}
		for _, keys := range curve.components {
			for _, key := range keys {
				if key.time >= start && key.time <= stop {
					times = append(times, key.time)
				}
			}
		}
		slices.Sort(times)
		times = slices.Compact(times)
		size := len(curve.components)
		if len(times) > budget/(size+1) {
			panic(fmt.Errorf("animation exceeds sample limit"))
		}
		budget -= len(times) * (size + 1)
		property := []string{"position", "quaternion", "scale"}[curve.property-1]
		track := AnimationTrack{Name: curve.target + "." + property, Type: "vector", Times: make([]float32, len(times)), Values: make([]float32, 0, len(times)*size)}
		if curve.property == 2 {
			track.Type = "quaternion"
		}
		if discrete {
			track.Interpolation = interpolateDiscrete
		}
		var previous [4]float64
		for i, time := range times {
			track.Times[i] = float32(time - start)
			var value [4]float64
			for component, keys := range curve.components {
				if discrete {
					index := sort.Search(len(keys), func(i int) bool { return keys[i].time > time })
					value[component] = keys[max(0, index-1)].value
				} else {
					value[component] = sampleAnimation(keys, time, baked || curve.linear)
				}
			}
			if curve.property == 1 {
				value[0] = -value[0]
			}
			if curve.property == 2 {
				value[1], value[2] = -value[1], -value[2]
				length := math.Sqrt(value[0]*value[0] + value[1]*value[1] + value[2]*value[2] + value[3]*value[3])
				if length == 0 {
					value[3], length = 1, 1
				}
				for c := range 4 {
					value[c] /= length
				}
				if i > 0 && previous[0]*value[0]+previous[1]*value[1]+previous[2]*value[2]+previous[3]*value[3] < 0 {
					for c := range 4 {
						value[c] = -value[c]
					}
				}
				previous = value
			}
			for _, component := range value[:size] {
				if !animationFinite(component) || !animationFinite(float64(float32(component))) {
					panic(fmt.Errorf("invalid %s curve values", property))
				}
				track.Values = append(track.Values, float32(component))
			}
		}
		animation.Tracks = append(animation.Tracks, optimizeAnimationTrack(track, size))
	}
	return animation
}

func explicitAnimationCurves(values map[string]any) []animationCurve {
	var curves []animationCurve
	for property, name := range []string{"m_PositionCurves", "m_RotationCurves", "m_ScaleCurves"} {
		entries, _ := values[name].([]any)
		for _, raw := range entries {
			entry := raw.(map[string]any)
			size := 3
			if property == 1 {
				size = 4
			}
			curve := animationCurve{target: animationTarget(crc32.ChecksumIEEE([]byte(entry["path"].(string)))), property: property + 1, components: make([][]animationKey, size)}
			data := entry["curve"].(map[string]any)
			keys := data["m_Curve"].([]any)
			if len(keys) == 0 {
				panic(fmt.Errorf("unsupported curve keys"))
			}
			for _, raw := range keys {
				key := raw.(map[string]any)
				time := animationFloat(key["time"])
				if !animationFinite(time) || key["weightedMode"] != nil && key["weightedMode"].(int64) != 0 ||
					len(curve.components[0]) > 0 && time <= curve.components[0][len(curve.components[0])-1].time {
					panic(fmt.Errorf("unsupported curve keys"))
				}
				for c, component := range []string{"x", "y", "z", "w"}[:size] {
					curve.components[c] = append(curve.components[c], animationKey{time,
						animationFloat(key["value"].(map[string]any)[component]),
						animationFloat(key["inSlope"].(map[string]any)[component]),
						animationFloat(key["outSlope"].(map[string]any)[component]), false})
				}
			}
			curves = append(curves, curve)
		}
	}
	return curves
}

func muscleAnimationCurves(packed, bindings map[string]any, start, stop float64) []animationCurve {
	stream := packed["m_StreamedClip"].(map[string]any)
	count := int(stream["curveCount"].(int64))
	if count < 0 || count > maxAssetSize/32 || stream["discreteCurveCount"] != nil && stream["discreteCurveCount"].(int64) != 0 {
		panic(fmt.Errorf("unsupported streamed curve count"))
	}
	channels := make([][]animationKey, count)
	words := stream["data"].([]any)
	var previousTime = math.Inf(-1)
	for pos := 0; pos < len(words); {
		if len(words)-pos < 2 {
			panic(fmt.Errorf("truncated streamed animation frame"))
		}
		time, n := animationFloat(words[pos]), int(words[pos+1].(int64))
		pos += 2
		if math.IsNaN(time) || time <= previousTime || n < 0 || n > (len(words)-pos)/5 || math.IsInf(time, 0) && n != 0 {
			panic(fmt.Errorf("invalid streamed animation frame"))
		}
		previousTime = time
		for range n {
			index := int(words[pos].(int64))
			if index < 0 || index >= count {
				panic(fmt.Errorf("invalid streamed animation channel"))
			}
			var coeff [4]float64
			for c := range 4 {
				coeff[c] = animationFloat(words[pos+1+c])
				if !animationFinite(coeff[c]) {
					panic(fmt.Errorf("invalid streamed animation coefficient"))
				}
			}
			pos += 5
			keys := channels[index]
			if len(keys) > 0 && time <= keys[len(keys)-1].time {
				panic(fmt.Errorf("duplicate streamed animation key"))
			}
			channels[index] = append(keys, animationKey{time: time, value: coeff[3], outSlope: coeff[2], inSlope: coeff[1], stepped: coeff[0] == 0 && coeff[1] == 0 && coeff[2] == 0})
		}
	}
	// Unity stores outgoing cubic coefficients; reconstruct the next key's incoming tangent.
	for _, keys := range channels {
		for i := len(keys) - 1; i > 0; i-- {
			a, b := keys[i-1], &keys[i]
			dx := math.Max(b.time-a.time, 0.0001)
			if a.stepped {
				b.inSlope = math.Inf(1)
			} else {
				b.inSlope = (3*(b.value-a.value) - 2*a.outSlope*dx - a.inSlope*dx*dx) / dx
			}
		}
	}
	dense := packed["m_DenseClip"].(map[string]any)
	denseCount, frames := int(dense["m_CurveCount"].(int64)), int(dense["m_FrameCount"].(int64))
	if denseCount < 0 || frames < 0 || denseCount > maxAssetSize/32 {
		panic(fmt.Errorf("invalid dense animation size"))
	}
	samples := dense["m_SampleArray"].([]any)
	if denseCount != 0 {
		rate, begin := animationFloat(dense["m_SampleRate"]), animationFloat(dense["m_BeginTime"])
		if !animationFinite(rate) || !animationFinite(begin) || rate <= 0 || frames <= 0 || frames > len(samples)/denseCount || frames*denseCount != len(samples) {
			panic(fmt.Errorf("invalid dense animation samples"))
		}
		for c := range denseCount {
			keys := make([]animationKey, frames)
			for frame := range frames {
				keys[frame] = animationKey{time: begin + float64(frame)/rate, value: animationFloat(samples[frame*denseCount+c])}
			}
			channels = append(channels, keys)
		}
	}
	constant := packed["m_ConstantClip"].(map[string]any)
	for _, raw := range constant["data"].([]any) {
		value := animationFloat(raw)
		channels = append(channels, []animationKey{{time: start, value: value}, {time: stop, value: value}})
	}
	var curves []animationCurve
	index := 0
	for _, raw := range bindings["genericBindings"].([]any) {
		binding := raw.(map[string]any)
		property := int(binding["attribute"].(int64))
		if binding["typeID"].(int64) != 4 || property < 1 || property > 3 || binding["customType"].(int64) != 0 || binding["isPPtrCurve"].(int64) != 0 || binding["isIntCurve"].(int64) != 0 {
			panic(fmt.Errorf("unsupported animation binding"))
		}
		size := 3
		if property == 2 {
			size = 4
		}
		if index+size > len(channels) {
			panic(fmt.Errorf("animation binding exceeds curve channels"))
		}
		components := channels[index : index+size]
		for _, keys := range components {
			if len(keys) == 0 {
				panic(fmt.Errorf("missing animation channel keys"))
			}
		}
		curves = append(curves, animationCurve{target: animationTarget(uint32(binding["path"].(int64))), property: property, components: components})
		index += size
	}
	if index != len(channels) {
		panic(fmt.Errorf("animation curve channels have no binding"))
	}
	slices.SortStableFunc(curves, func(a, b animationCurve) int { return a.property - b.property })
	return curves
}

func sampleAnimation(keys []animationKey, time float64, linear bool) float64 {
	if time <= keys[0].time {
		return keys[0].value
	}
	next := sort.Search(len(keys), func(i int) bool { return keys[i].time > time })
	if next == len(keys) {
		return keys[len(keys)-1].value
	}
	a, b := keys[next-1], keys[next]
	duration := b.time - a.time
	t := (time - a.time) / duration
	if linear {
		return a.value + (b.value-a.value)*t
	}
	if !animationFinite(a.outSlope) || !animationFinite(b.inSlope) {
		return a.value
	}
	t2, t3 := t*t, t*t*t
	return (2*t3-3*t2+1)*a.value + (t3-2*t2+t)*duration*a.outSlope +
		(-2*t3+3*t2)*b.value + (t3-t2)*duration*b.inSlope
}

func optimizeAnimationTrack(track AnimationTrack, size int) AnimationTrack {
	times, values := track.Times, track.Values
	keep := []int{0}
	for i := 1; i < len(times)-1; i++ {
		if times[i] == times[i+1] || i == 1 && times[i] == times[0] {
			continue
		}
		for c := range size {
			value := values[i*size+c]
			if value != values[(i-1)*size+c] || value != values[(i+1)*size+c] {
				keep = append(keep, i)
				break
			}
		}
	}
	if len(times) > 1 {
		keep = append(keep, len(times)-1)
	}
	for write, read := range keep {
		track.Times[write] = times[read]
		copy(track.Values[write*size:(write+1)*size], values[read*size:(read+1)*size])
	}
	track.Times = track.Times[:len(keep)]
	track.Values = track.Values[:len(keep)*size]
	return track
}
