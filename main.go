package main

import (
	"flag"
	"log"
	"time"

	"github.com/seraphliu/harbor-replicator/dockerclient"
	"github.com/seraphliu/harbor-replicator/harborclient"
	"fmt"
	"os"
)

var (
	harbor   = flag.String("harbor", "", "harbor registry server address")
	user     = flag.String("huser", "", "user for harbor")
	pass     = flag.String("hpass", "", "password for harbor")
	insecure = flag.Bool("insecure", false, "using http:// scheme for harbor")
	rhost    = flag.String("remote", "", "remote registry")
	ruser    = flag.String("remote_user", "", "user for remote registry")
	rpass    = flag.String("remote_pass", "", "password for remote registry")
	project  = flag.String("project", "", "filter projects")
	since    = flag.Duration("since", 720*time.Hour, "sync since some time duration ago")
)

func harborStatusChecker(harbor string, user, pass string, project string, since time.Time, insecure bool) <-chan *harborclient.Event {
	out := make(chan *harborclient.Event, 1)
	time.ParseDuration("1d")
	go func() {
		c := harborclient.NewHarborClient(harbor, user, pass, project, insecure)

		for {
			err := c.RefreshRepos()
			if err != nil {
				log.Println(err)
			}

			for _, k := range c.GetRepoNames() {
				tags, err := c.RefreshRepoTags(k)
				if err != nil {
					log.Println(err)
					continue
				}
				for t := range tags {
					if tags[t].After(since) {
						out <- &harborclient.Event{
							Repo: k,
							Tag:  t,
						}
					}
				}
			}
			time.Sleep(3 * time.Second)
		}
	}()

	return out
}

func main() {
	flag.Parse()

	if *harbor == "" {
		fmt.Println("empty harbor server address")
		os.Exit(1)
	}
	if *rhost == "" {
		fmt.Println("empty remote server address")
		os.Exit(1)
	}

	watchTime := time.Now().Add(- *since)

	out := harborStatusChecker(*harbor, *user, *pass, *project, watchTime, *insecure)

	local := dockerclient.NewDockerClient(*harbor, *user, *pass)
	if err := local.Login(); err != nil {
		log.Fatalf("login to harbor %s error: %v", *harbor, err)
	}
	remote := dockerclient.NewDockerClient(*rhost, *ruser, *rpass)
	if err := remote.Login(); err != nil {
		log.Fatalf("login to remote registry %s error: %v", *rhost, err)
	}

	for e := range out {
		log.Printf("detect new repo: %s:%s\n", e.Repo, e.Tag)
		done := make(chan struct{})
		go func() {
			if err := local.Pull(e.Repo, e.Tag); err != nil {
				log.Println("pull image error: ", err)
				return
			}
			if err := remote.Tag(*harbor, e.Repo, e.Tag); err != nil {
				log.Println("tag image error: ", err)
				return
			}
			if err := remote.Push(e.Repo, e.Tag); err != nil {
				log.Println("push image error: ", err)
				return
			}
			done <- struct{}{}
		}()
		select {
		case <-time.After(300 * time.Second):
			log.Println("replication time out, skip")
		case <-done:
			log.Println("replicate to remote registry successfully")
		}
	}
}
