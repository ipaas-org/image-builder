package harbor_test

import (
	"context"
	"log"
	"os"
	"testing"

	"github.com/ipaas-org/image-builder/config"
	"github.com/ipaas-org/image-builder/pkg/logger"
	"github.com/ipaas-org/image-builder/providers/registry/harbor"
)

func newHarborRegistry() *harbor.HarborClient {
	conf, err := config.NewConfig("../../../../")
	if err != nil {
		log.Fatalln(err)
	}

	l := logger.NewLogger("debug", "text")
	l.Debug("initizalized logger")
	l.Infof("conf: %+v\n", conf)

	r, err := harbor.NewHarborRegistry(conf.Services.Registries[0].ServerAddress, os.Getenv("REGISTRY_USERNAME"), os.Getenv("REGISTRY_PASSWORD"), os.Getenv("REGISTRY_PULL_USERNAME"), os.Getenv("REGISTRY_PULL_PASSWORD"))
	if err != nil {
		l.Fatalln(err)
	}
	return r
}

func TestCompleteTransaction(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	h := newHarborRegistry()

	// localImageID, userCode, appName string
	imageId := "3f57d9401f8d" //busybox
	userCode := "us-test"
	appName := "busybox"
	tag, err := h.TagImage(ctx, imageId, userCode, appName)
	if err != nil {
		t.Fatal(err)
	}

	err = h.PushImage(ctx, tag)
	if err != nil {
		t.Fatal(err)
	}
}
