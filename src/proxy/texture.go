package proxy

import (
	"bytes"
	"fmt"
	"image"
	"image/png"
	"io"
	"net/http"
	"path"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/kvarenzn/ssm/decoders/astc"
	unitylog "github.com/kvarenzn/ssm/log"
	"github.com/kvarenzn/ssm/uni"
)

func init() {
	// The upstream reader exits on malformed data; keep decode failures local to the request.
	unitylog.SetBeforeDie(func() { panic("invalid Unity asset") })
}

func (h *handler) handleTexture(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseUint(chi.URLParam(r, "id"), 10, 32)
	if err != nil {
		http.Error(w, "Invalid resource ID", http.StatusBadRequest)
		return
	}
	hash, ok := h.www.ResourceHash("ios", uint32(id))
	if !ok {
		http.Error(w, "Resource not found", http.StatusNotFound)
		return
	}
	file, err := h.www.OpenResourceByHash(hash)
	if err != nil {
		http.Error(w, "Asset not found", http.StatusNotFound)
		return
	}
	defer file.Close()
	data, err := io.ReadAll(file)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	texture, err := texturePNG(data, hash)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnprocessableEntity)
		return
	}
	w.Header().Set("Content-Type", "image/png")
	http.ServeContent(w, r, hash+".png", time.Time{}, bytes.NewReader(texture))
}

func texturePNG(data []byte, name string) (result []byte, err error) {
	defer func() {
		if failure := recover(); failure != nil {
			result = nil
			err = fmt.Errorf("failed to decode Unity texture: %v", failure)
		}
	}()
	reader, err := uni.NewFileReader(data, name)
	if err != nil {
		return nil, err
	}
	manager := uni.NewAssetsManager()
	switch reader.FileType {
	case uni.FileTypeBundleFile:
		err = manager.LoadBundle(reader, "")
	case uni.FileTypeAssetsFile:
		reader.SeekTo(0)
		err = manager.LoadAssets(reader, name, "", nil)
	default:
		return nil, fmt.Errorf("unsupported Unity asset file")
	}
	if err != nil {
		return nil, err
	}
	for _, file := range manager.AssetFiles {
		for _, info := range file.ObjectInfos {
			if info.ClassID != uni.ClassIDTexture2D {
				continue
			}
			texture := uni.NewTexture2D(uni.NewObjectReader(file.Reader.BinaryReader, file, info))
			img, err := decodeTexture(texture)
			if err != nil {
				return nil, err
			}
			var encoded bytes.Buffer
			if err := png.Encode(&encoded, img); err != nil {
				return nil, err
			}
			return encoded.Bytes(), nil
		}
	}
	return nil, fmt.Errorf("asset contains no Texture2D")
}

func decodeTexture(texture *uni.Texture2D) (image.Image, error) {
	if texture.Width <= 0 || texture.Height <= 0 || texture.ImageData.Size <= 0 {
		return nil, fmt.Errorf("invalid texture dimensions or data")
	}
	if texture.ImageData.NeedSearch {
		// Resolve streams from this bundle, without the upstream reader's filesystem fallback.
		reader := texture.AssetFile.AssetsManager.ResourceFileReaders[path.Base(texture.ImageData.Path)]
		if reader == nil {
			return nil, fmt.Errorf("streamed texture data not found")
		}
		texture.ImageData = uni.NewResourceReader(reader, texture.ImageData.Offset, texture.ImageData.Size)
	}
	data := texture.ImageData
	if data.Offset < 0 || data.Offset > data.Reader.Len() || data.Size > data.Reader.Len()-data.Offset {
		return nil, fmt.Errorf("texture data outside asset")
	}
	// DecodeTexture2D omits ASTC 5x5 and the older RGBA aliases used by iOS bundles.
	blocks := [...]int{4, 5, 6, 8, 10, 12}
	format := texture.Format
	if format >= uni.ASTC_RGBA_4x4 && format <= uni.ASTC_RGBA_12x12 {
		format -= uni.ASTC_RGBA_4x4 - uni.ASTC_RGB_4x4
	}
	if format >= uni.ASTC_RGB_4x4 && format <= uni.ASTC_RGB_12x12 {
		block := blocks[format-uni.ASTC_RGB_4x4]
		return astc.Decode(texture.ImageData.GetData(), int(texture.Width), int(texture.Height), block, block)
	}
	return uni.DecodeTexture2D(texture)
}
