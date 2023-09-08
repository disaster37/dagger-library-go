package image

import (
	"context"
	"os"
	"testing"

	"dagger.io/dagger"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

func TestBuildImage(t *testing.T) {
	var err error
	logrus.SetLevel(logrus.DebugLevel)

	ctx := context.Background()

	// initialize Dagger client
	client, err := dagger.Connect(ctx, dagger.WithLogOutput(os.Stdout))
	if err != nil {
		panic(err)
	}
	defer client.Close()

	buildOption := &BuildImageOption{
		RegistryName: "docker.io",
		Name:         "test",
		Tag:          "test",
		PathContext:  "testdata",
	}

	err = BuildImage(ctx, client, buildOption)
	assert.NoError(t, err)
}
