package e2e

import (
	"context"
	"fmt"
	"net"
	"os"
	"os/exec"
	"strconv"
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/e2e-framework/klient/k8s/resources"
	"sigs.k8s.io/e2e-framework/klient/wait"
	"sigs.k8s.io/e2e-framework/klient/wait/conditions"
	"sigs.k8s.io/e2e-framework/pkg/env"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/support"
	"sigs.k8s.io/e2e-framework/support/kwok"
)

var (
	testEnv  env.Environment
	etcdPort = getUnusedPort()
	endpoint = fmt.Sprintf("127.0.0.1:%d", etcdPort)
)

func TestMain(m *testing.M) {
	cfg, _ := envconf.NewFromFlags()
	testEnv = env.NewWithConfig(cfg)
	clusterName := envconf.RandomName("augerctl", 32)

	provider := kwok.NewProvider().WithName(clusterName)

	testEnv.Setup(
		Build(),
		CreateCluster(provider,
			"--etcd-port="+strconv.Itoa(etcdPort),
		),
	)

	testEnv.Finish(
		DeleteCluster(provider),
	)

	os.Exit(testEnv.Run(m))
}

func CreateCluster(p support.E2EClusterProvider, args ...string) env.Func {
	return func(ctx context.Context, cfg *envconf.Config) (context.Context, error) {
		kubecfg, err := p.Create(ctx, args...)
		if err != nil {
			return ctx, err
		}

		cfg.WithKubeconfigFile(kubecfg)

		r, err := resources.New(cfg.Client().RESTConfig())
		if err != nil {
			return ctx, err
		}
		err = wait.For(
			conditions.New(r).ResourceListN(&corev1.ServiceAccountList{}, 1),
			wait.WithTimeout(20*time.Minute),
			wait.WithContext(ctx),
		)
		if err != nil {
			return ctx, err
		}
		return ctx, nil
	}
}

func DeleteCluster(p support.E2EClusterProvider) env.Func {
	return func(ctx context.Context, cfg *envconf.Config) (context.Context, error) {
		err := p.Destroy(ctx)
		if err != nil {
			return ctx, err
		}
		return ctx, nil
	}
}

var augerctl = "../../build/augerctl"

func Build() env.Func {
	return func(ctx context.Context, cfg *envconf.Config) (context.Context, error) {
		err := exec.CommandContext(ctx, "make", "-C", "../..", "build").Run()
		if err != nil {
			return ctx, err
		}
		return ctx, nil
	}
}

func getUnusedPort() int {
	listener, err := net.Listen("tcp", ":0")
	if err != nil {
		panic(err)
	}
	defer listener.Close()
	return listener.Addr().(*net.TCPAddr).Port
}
