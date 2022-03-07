# code-code-server
code-server launcher with vscode devcontainer config

## Installation

Use `go install` to install code-code-server.

```bash
go install github.com/ar90n/code-code-server/cmd/code@latest
```

## Usage
Change directory to the directory you want to serve. And run `code .` to build image and start container. The logs of these operations are following.

```bash
$ cd code-code-server
$ ls .devcontainer
Dockerfile              devcontainer.json
$ code .
[+] Building 0.4s (10/10) FINISHED
 => [internal] load build definition from Dockerfile                                                                                                                                                     0.0s
 => => transferring dockerfile: 1.64kB                                                                                                                                                                   0.0s
 => [internal] load .dockerignore                                                                                                                                                                        0.0s
 => => transferring context: 2B                                                                                                                                                                          0.0s
 => [internal] load metadata for mcr.microsoft.com/vscode/devcontainers/go:0-1-bullseye                                                                                                                  0.3s
 => [1/6] FROM mcr.microsoft.com/vscode/devcontainers/go:0-1-bullseye@sha256:4cae6b242e4c6357f3242c0c6c70987cf56ad42e7c3ae925ea3ad3525925f891                                                            0.0s
 => CACHED [2/6] RUN if [ "lts/*" != "none" ]; then su vscode -c "umask 0002 && . /usr/local/share/nvm/nvm.sh && nvm install lts/* 2>&1"; fi                                                             0.0s
 => CACHED [3/6] RUN curl -fsSL https://code-server.dev/install.sh | sh                                                                                                                                  0.0s
 => CACHED [4/6] RUN mkdir -p /opt/code-server                                                                                                                                                           0.0s
 => CACHED [5/6] RUN { echo '#!/bin/bash'; echo 'set -e'; echo 'set -x'; echo ''; echo 'code-server --install-extension golang.Go'; echo 'echo "auth: none" > /tmp/config.yml'; echo 'code-server --con  0.0s
 => CACHED [6/6] RUN chmod +x /opt/code-server/entrypoint.sh                                                                                                                                             0.0s
 => exporting to image                                                                                                                                                                                   0.0s
 => => exporting layers                                                                                                                                                                                  0.0s
 => => writing image sha256:1881c619ccb5ea4c4b90ba9466a0ae3255eba897159ca39e87494c535f7099a8                                                                                                             0.0s
 => => naming to docker.io/library/go_code_coder_server                                                                                                                                                  0.0s
2022/03/07 22:24:41 ==============================================================================================
2022/03/07 22:24:41 Code Server running at http://tororo.local:58818/?folder=/workspace/code-code-server
2022/03/07 22:24:41 ==============================================================================================
+ code-server --install-extension golang.Go
[2022-03-07T13:24:43.113Z] info  Wrote default config file to ~/.config/code-server/config.yaml
Installing extensions...
Installing extension 'golang.go'...
Extension 'golang.go' v0.31.1 was successfully installed.
+ echo 'auth: none'
+ code-server --config /tmp/config.yml --bind-addr 0.0.0.0:8080
[2022-03-07T13:24:53.607Z] info  code-server 4.1.0 9e620e90f53fb91338a2ba1aaa2e556d42ae52d5
[2022-03-07T13:24:53.609Z] info  Using user-data-dir ~/.local/share/code-server
[2022-03-07T13:24:53.642Z] info  Using config file /tmp/config.yml
[2022-03-07T13:24:53.643Z] info  HTTP server listening on http://0.0.0.0:8080/
[2022-03-07T13:24:53.643Z] info    - Authentication is disabled
[2022-03-07T13:24:53.643Z] info    - Not serving HTTPS
```

And you can access to the code server by the above URL which is `http://tororo.local:58818/?folder=/workspace/code-code-server`.

![スクリーンショット 2022-03-07 22 29 31](https://user-images.githubusercontent.com/2285892/157044688-6c1ed4e2-1426-459e-b489-644b6ec9d25b.png)

## Contributing
Pull requests are welcome. For major changes, please open an issue first to discuss what you would like to change.

Please make sure to update tests as appropriate.

## License
[Apache-2.0](https://www.apache.org/licenses/LICENSE-2.0)
