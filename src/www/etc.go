package www

import (
	"fmt"
	"net"
	"net/http"

	"example.com/brave-revival/src/proto/proto"
)

func actionlog(w http.ResponseWriter, _ *http.Request) {
	writeProto(w, http.StatusOK, &proto.Empty{})
}

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
			Realtime: "wss://" + chatHost,
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
	writeProto(w, http.StatusOK, getHandler(r).master)
}

func resourceList(w http.ResponseWriter, r *http.Request) {
	res := getHandler(r).resources
	w.Header().Set("x-enish-app-resource-cnt", fmt.Sprint(len(res.Resource)))
	writeProto(w, http.StatusOK, res)
}
