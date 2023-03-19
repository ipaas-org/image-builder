package main

import (
	"fmt"
	"log"

	"github.com/labstack/echo/v4"
	"github.com/vano2903/image-builder/config"
	"github.com/vano2903/image-builder/controller"
	"github.com/vano2903/image-builder/model"
	"github.com/vano2903/image-builder/pkg/logger"
	"github.com/vano2903/image-builder/providers/builders/nixpacks"
	"github.com/vano2903/image-builder/providers/connectors/github"
	// "github.com/vano2903/image-builder/repo/mock"
)

func main() {
	conf, err := config.NewConfig()
	if err != nil {
		panic(err)
	}

	l := logger.NewLogger(conf.Log.Level, conf.Log.Type)
	l.Debug("initizalized logger")

	// if conf.Database.Driver != "mock" {
	// 	log.Fatal("only mock database is supported in this example")
	// }

	// //creating the instances for the application
	// repo := mock.NewRepo()

	//creating the controller
	c := controller.NewBuilderController(l)

	l.Info(conf)
	for _, providerInfo := range conf.Services.Connectors {
		switch providerInfo.Name {
		case model.ConnectorGithub:
			g := github.NewGithubConnector(providerInfo.DownloadDirectory, fmt.Sprintf("ipaas-%s-%s", conf.App.Name, conf.App.Version), "", l)
			c.AddConnector(model.ConnectorGithub, g)
			l.Infof("succesfully added %s as downloader", providerInfo.Name)

		default:
			l.Errorf("provider %s not supported", providerInfo.Name)
		}
	}

	if len(conf.Services.Builders) == 0 {
		log.Fatal("no builders specified")
	}

	if conf.Services.Builders[0].Name != model.DownloaderNixpacks {
		log.Fatal("only nixpacks builder is supported in the app version:", conf.App.Version)
	}

	nix := nixpacks.NewNixPackBuilder(conf.Services.Builders[0].RegistryUri)

	c.AddBuilder(model.DownloaderNixpacks, nix)

	//creating the http server
	e := echo.New()

	//starting the server
	e.Logger.Fatal(e.Start(":" + "8080")) //conf.HTTP.Port))
}

// func main() {
// 	builder := builder.NixPackBuilder{}

// 	plan, err := builder.Plan(context.Background(), "./testing")
// 	fmt.Println(plan, err)

// 	fmt.Println(builder.Build(context.Background(), plan, "./testing"))
// }
