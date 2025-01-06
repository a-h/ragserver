{
  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-24.11";
    gitignore = {
      url = "github:hercules-ci/gitignore.nix";
      inputs.nixpkgs.follows = "nixpkgs";
    };
    xc = {
      url = "github:joerdav/xc";
      inputs.nixpkgs.follows = "nixpkgs";
    };
  };

  outputs = { self, nixpkgs, gitignore, xc }:
    let
      allSystems = [
        "x86_64-linux" # 64-bit Intel/AMD Linux
        "aarch64-linux" # 64-bit ARM Linux
        "x86_64-darwin" # 64-bit Intel macOS
        "aarch64-darwin" # 64-bit ARM macOS
      ];

      forAllSystems = f: nixpkgs.lib.genAttrs allSystems (system: f {
        system = system;
        pkgs = import nixpkgs {
          inherit system;
          overlays = [
            (final: prev: {
              rqlite = prev.rqlite.overrideAttrs (oldAttrs: {
                version = "8.34.2";
                src = prev.fetchFromGitHub {
                  owner = "rqlite";
                  repo = "rqlite";
                  rev = "v8.34.2";
                  hash = "sha256-+/D5sHDzhBmF6C1JKGaEJSVdcIyU8o9n0qc1/xEoxjo=";
                };
                vendorHash = "sha256-v30TFML8RBn02LaNDQ0LBbhJduQUZDEBUCSSDwW2Ixo=";
              });
              sqlite-vec = prev.sqlite-vec.overrideAttrs (oldAttrs: {
                version = "0.1.6";
                src = prev.fetchFromGitHub {
                  owner = "asg017";
                  repo = "sqlite-vec";
                  rev = "v0.1.6";
                  sha256 = "sha256-CgeSoRoQRMb/V+RzU5NQuIk/3OonYjAfolWD2hqNuXU=";
                };
                installPhase = ''
                  runHook preInstall

                  # I've customised this to only install the shared library.
                  # Otherwise, rqlite tries to load the static library (and fails).
                  install -Dm444 -t "$out/lib" \
                    "dist/vec0${prev.stdenv.hostPlatform.extensions.sharedLibrary}"

                  runHook postInstall
                '';
              });
              xc = xc.packages.${system}.xc;
            })
          ];
        };
      });

      # Build app.
      app = { name, pkgs, system }: pkgs.buildGoModule {
        name = name;
        pname = name;
        src = gitignore.lib.gitignoreSource ./.;
        go = pkgs.go;
        subPackages = [ "cmd/${name}" ];
        vendorHash = null; # Use vendored dependencies.
        CGO_ENABLED = 0;
        flags = [
          "-trimpath"
        ];
        ldflags = [
          "-s"
          "-w"
          "-extldflags -static"
        ];
      };

      # Build Docker containers.
      dockerUser = pkgs: pkgs.runCommand "user" { } ''
        mkdir -p $out/etc
        echo "user:x:1000:1000:user:/home/user:/bin/false" > $out/etc/passwd
        echo "user:x:1000:" > $out/etc/group
        echo "user:!:1::::::" > $out/etc/shadow
      '';
      dockerImage = { name, pkgs, system }: pkgs.dockerTools.buildImage {
        name = name;
        tag = "latest";

        copyToRoot = [
          # Remove coreutils and bash for a smaller container.
          pkgs.coreutils
          pkgs.bash
          (dockerUser pkgs)
          (app { inherit name pkgs system; })
        ];
        config = {
          Cmd = [ "ragserver" "serve" ];
          User = "user:user";
          ExposedPorts = {
            "9020/tcp" = { };
          };
        };
      };

      rqliteDockerImage = { pkgs, system }: pkgs.dockerTools.buildImage {
        name = "rqlite";
        tag = "latest";

        copyToRoot = [
          pkgs.coreutils
          pkgs.bash
          (dockerUser pkgs)
          pkgs.rqlite
          pkgs.sqlite-vec
        ];
        config = {
          Entrypoint = [ "rqlited" "-http-addr" "0.0.0.0:4001" "-http-adv-addr" "rqlite.ragserver.svc.cluster.local:4001" "-raft-addr" "0.0.0.0:4002" "-raft-adv-addr" "rqlite.ragserver.svc.cluster.local:4002" "-auth" "/mnt/rqlite/auth.json" "-extensions-path" "${pkgs.sqlite-vec}/lib" "/mnt/data" ];
          User = "user:user";
          ExposedPorts = {
            "4001/tcp" = { };
            "4002/tcp" = { };
            "4003/tcp" = { };
          };
          Volumes = {
            "/rqlite/file" = { };
          };
        };
      };

      # Development tools used.
      devTools = { system, pkgs }: [
        pkgs.sqlite # Full text database.
        pkgs.crane
        pkgs.gh
        pkgs.git
        pkgs.go
        pkgs.gopls
        pkgs.xc
        pkgs.envsubst
        pkgs.kubectl
        pkgs.k9s
        # Database tools.
        pkgs.rqlite # Distributed sqlite.
        pkgs.go-migrate # Migrate database schema.
        # Vector extension.
        pkgs.sqlite-vec
      ];

      name = "ragserver";
    in
    {
      # `nix build` builds the app.
      # `nix build .#docker-image` builds the Docker container.
      packages = forAllSystems ({ system, pkgs }: {
        default = app { name = name; pkgs = pkgs; system = system; };
        docker-image = dockerImage { name = name; pkgs = pkgs; system = system; };
        rqlite-docker-image = rqliteDockerImage { pkgs = pkgs; system = system; };
      });
      # `nix develop` provides a shell containing required tools.
      devShells = forAllSystems ({ system, pkgs }: {
        default = pkgs.mkShell {
          buildInputs = (devTools { system = system; pkgs = pkgs; });
          shellHook = ''
            export SQLITE_VEC_PATH=${pkgs.sqlite-vec}/lib
            echo "SQLITE_VEC_PATH is set to $SQLITE_VEC_PATH"
          '';
        };
      });
    };
}
