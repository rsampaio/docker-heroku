Docker-heroku project annotations
=================================

Introduction
------------

### The goals of the project are: 
	- Given an app name and release version

	- Grab the slug blob

	- Unpack

	- Map the unpacked slug to a /app inside a container
	
	- Run the container with the commands of a specified process type

Summary
-------

- Project entirely in Go

- Using slug API V3

- Only external dependencies are docker and go-dockerclient 
	- (I'm using my own fork because of some fixes needed to work with latest Docker version)

- Go dockerclient lib has some issues, use internal Docker 
	- i.e: docker.Config and docker.HostConfig which are tricky to figure.

Installation
------------

> go get -v github.com/rsampaio/docker-heroku/...

Usage
-----

> docker-heroku -username rodrigo.vaz@xxx.com -token a1b2c3d4e5f6 -app heroku-app-name -release 3 -process web


Problems and what I've learned
------------------------------

- Slugs and Releases API where a little tricky to figure how to use version 3 
	- "Accept: application/vnc.heroku+json; version=3"

- Go archive/tar and compress/gzip library

- Using the go-dockerclient to mount volumes is kind of a hack due to the internal structs of docker
	> volumes := make(map[string]struct{})
	> 
	> volumes["/app"] = struct{}{}
	> 
	> err = dClient.StartContainer(container.ID, &docker.HostConfig{Binds: []string{cwd + "/app:/app"}})

- Still prefered go-dockerclient because it can communicate with the default unix socket at /var/run/docker.sock
	- Tried to avoid any docker configuration tweak like listening on tcp port instead of unix domain

- Used a runtime image based on my own project (github.com/rsampaio/dockstep)
	- Basically a dockerfile that install the build deps and clone all buildpacks
	- Could be generated by cedar.sh stack-image 

Possible improvements
---------------------

- Maybe a pre-made image on the docker index on heroku namespace would be nice
	- This image could be built by cedar.sh (or a variation like cedar_docker.sh)

- A progress bar or progress indication while downloading the slug 
	- The slug size is already listed on the API 

- Use the upcoming ONBUILD feature of docker
	- This feature will run ONBUILD commands when you build a new image based on the image built with ONBUILD commands

- Tag the container for each process type ex: web, worker, etc

- Inject app dir on the container since mapping is only possible for 1 container at a time