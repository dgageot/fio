package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strconv"
	"syscall"

	"golang.org/x/sync/errgroup"
)

const (
	imageName = "fio"
	testName  = "test-fio"
)

func main() {
	count := 1
	if len(os.Args) > 1 {
		count, err := strconv.Atoi(os.Args[1])
		if err != nil {
			panic(err)
		}

		fmt.Printf("Running %d tests in concurrently\n", count)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	catchCtrlC(cancel)

	var errs errgroup.Group
	for i := 0; i < count; i++ {
		testName := fmt.Sprintf("%s-%d", testName, i)

		errs.Go(func() error {
			return run(ctx, imageName, testName)
		})
	}

	if err := errs.Wait(); err != nil {
		panic(err)
	}
}

func run(ctx context.Context, imageName, testName string) error {
	if err := buildImage(ctx, imageName); err != nil {
		return fmt.Errorf("unable to build the docker image: %v", err)
	}

	wd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("unable to get working dir: %v", err)
	}

	runOneTest(ctx, wd, imageName, testName)
	remove(testName)
	return err
}

func runOneTest(ctx context.Context, wd, imageName, testName string) error {
	if err := createFolder(testName); err != nil {
		return fmt.Errorf("unable to create main folder: %v", err)
	}

	cmdRun, err := startDocker(ctx, "run", "--rm", "--init", "--name", testName, "-v", filepath.Join(wd, "docker_host_volume")+":/datavolume", imageName, "sh", "-c", "for i in {1..14400}; do echo $i; ls \"/datavolume/${name}\"; sleep 1; done")
	if err != nil {
		return fmt.Errorf("unable to docker run: %v", err)
	}

	for {
		cmd := exec.CommandContext(ctx, "docker", "inspect", testName)
		if err := cmd.Run(); err == nil {
			break
		}
	}

	_, err = startDocker(ctx, "logs", "-f", testName)
	if err != nil {
		return fmt.Errorf("unable to docker logs: %v", err)
	}

	cmdExec, err := startDocker(ctx, "exec", testName, "fio", "--name", testName, "--directory", filepath.Join("datavolume", testName), "--numjobs=3", "--size=8388608", "--time_based", "--runtime=14400s", "--ramp_time=2s", "--ioengine=libaio", "--direct=1", "--verify=0", "--bs=4096", "--iodepth=256", "--rw=read", "--group_reporting=1", "--kb_base=1024", "--unit_base=8")
	if err != nil {
		return fmt.Errorf("unable to docker exec: %v", err)
	}

	cmdExec.Wait()
	cmdRun.Wait()

	return nil
}

func remove(testName string) {
	cmd := exec.Command("docker", "rm", "-f", testName)
	_ = cmd.Run()
}

func startDocker(ctx context.Context, args ...string) (*exec.Cmd, error) {
	cmd := exec.CommandContext(ctx, "/usr/local/bin/com.docker.cli", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd, cmd.Start()
}

func buildImage(ctx context.Context, imageName string) error {
	cmd, err := startDocker(ctx, "build", "-t", imageName, "build")
	if err != nil {
		return err
	}
	return cmd.Wait()
}

func createFolder(testName string) error {
	folderPath := filepath.Join("docker_host_volume", testName)
	return os.MkdirAll(folderPath, 0755)
}

// catchCtrlC calls the `cancel` callback if the process is interrupted, for eg. with ctrl-c.
func catchCtrlC(cancel context.CancelFunc) {
	signals := make(chan os.Signal, 2)
	signal.Notify(signals,
		os.Interrupt,
		syscall.SIGTERM,
		syscall.SIGINT,
		syscall.SIGPIPE,
	)

	go func() {
		<-signals
		signal.Stop(signals)
		cancel()
	}()
}
