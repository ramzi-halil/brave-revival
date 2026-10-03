package www

import (
	"fmt"
	"net"
	"net/http"

	"example.com/brave-revival/src/proto/proto"
	"github.com/go-chi/chi/v5"
	pb "google.golang.org/protobuf/proto"
)

func etc(w http.ResponseWriter, r *http.Request) {
	config := getHandler(r).config
	chatHost := net.JoinHostPort(config.AdvertiseHost, fmt.Sprint(config.ChatPort))
	writeProto(w, http.StatusOK, &proto.Etc{
		Host: &proto.Host{
			Photon:   config.AdvertiseHost,
			Chat:     chatHost,
			Party:    net.JoinHostPort(config.AdvertiseHost, fmt.Sprint(config.PartyPort)),
			Notify:   net.JoinHostPort(config.AdvertiseHost, fmt.Sprint(config.NotifyPort)),
			Gvg:      config.AdvertiseHost,
			Realtime: "ws://" + chatHost,
			GvgHosts: 1,
		},
		Language:       "ja",
		Environment:    "production",
		Gmt:            9.0,
		StoreReviewUrl: "https://example.com/",
		Revision: &proto.Revision{
			Terms:         4,
			PrivacyPolicy: 4,
		},
	})
}

func masterAll(w http.ResponseWriter, r *http.Request) {
	handler := getHandler(r)
	writeProtoStreamed(w, http.StatusOK, func(stream func(pb.Message) error) error {
		handler.masterLock.Lock()
		defer handler.masterLock.Unlock()
		return stream(handler.master)
	})
}

func resourceList(w http.ResponseWriter, r *http.Request) {
	handler := getHandler(r)
	platform := chi.URLParam(r, "os")
	writeProtoStreamed(w, http.StatusOK, func(stream func(pb.Message) error) error {
		handler.masterLock.Lock()
		defer handler.masterLock.Unlock()
		res := handler.resources[platform]
		w.Header().Set("x-enish-app-resource-cnt", fmt.Sprint(len(res.Resource)))
		return stream(res)
	})
}

func news(w http.ResponseWriter, _ *http.Request) {
	w.Write([]byte("<h1>Hello</h1>"))
}
