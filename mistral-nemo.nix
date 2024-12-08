{ pkgs ? import <nixpkgs> {} }:

let
  # List of blob files with URLs and corresponding hashes.
  blob_0 = pkgs.fetchurl {
    curlOptsList = ["-L" "-H" "Accept:application/octet-stream"];
    url = "https://registry.ollama.ai/v2/library/mistral-nemo/blobs/sha256:b559938ab7a0392fc9ea9675b82280f2a15669ec3e0e0fc491c9cb0a7681cf94";
    hash = "sha256-tVmTiregOS/J6pZ1uCKA8qFWaew+Dg/EkcnLCnaBz5Q=";
  };
  blob_1 = pkgs.fetchurl {
    curlOptsList = ["-L" "-H" "Accept:application/octet-stream"];
    url = "https://registry.ollama.ai/v2/library/mistral-nemo/blobs/sha256:f023d1ce0e55d0dcdeaf70ad81555c2a20822ed607a7abd8de3c3131360f5f0a";
    hash = "sha256-8CPRzg5V0Nzer3CtgVVcKiCCLtYHp6vY3jwxMTYPXwo=";
  };
  blob_2 = pkgs.fetchurl {
    curlOptsList = ["-L" "-H" "Accept:application/octet-stream"];
    url = "https://registry.ollama.ai/v2/library/mistral-nemo/blobs/sha256:43070e2d4e532684de521b885f385d0841030efa2b1a20bafb76133a5e1379c1";
    hash = "sha256-QwcOLU5TJoTeUhuIXzhdCEEDDvorGiC6+3YTOl4TecE=";
  };
  blob_3 = pkgs.fetchurl {
    curlOptsList = ["-L" "-H" "Accept:application/octet-stream"];
    url = "https://registry.ollama.ai/v2/library/mistral-nemo/blobs/sha256:ed11eda7790d05b49395598a42b155812b17e263214292f7b87d15e14003d337";
    hash = "sha256-7RHtp3kNBbSTlVmKQrFVgSsX4mMhQpL3uH0V4UAD0zc=";
  };
  blob_4 = pkgs.fetchurl {
    curlOptsList = ["-L" "-H" "Accept:application/octet-stream"];
    url = "https://registry.ollama.ai/v2/library/mistral-nemo/blobs/sha256:65d37de20e5951c7434ad4230c51a4d5be99b8cb7407d2135074d82c40b44b45";
    hash = "sha256-ZdN94g5ZUcdDStQjDFGk1b6ZuMt0B9ITUHTYLEC0S0U=";
  };

  # Fetch the manifest file.
  manifestFile = pkgs.fetchurl {
    curlOptsList = ["-L" "-H" "Accept:application/octet-stream"];
    url = "https://registry.ollama.ai/v2/library/mistral-nemo/manifests/latest";
    hash = "sha256-szHCV7DSbF9bYDkENYWgSyifTiZ7NEOFm4IuM4j+ZIs=";
  };
in
  # Use symlinkJoin to create the final symlinked structure.
  pkgs.symlinkJoin {
    name = "models";

    # Paths from both blobs and the manifest file.
    paths = [ ];

    # Add a postBuild step to arrange the structure.
    postBuild = ''
      # Move blob files to the blobs directory.
      mkdir -p $out/blobs
      ln -s ${blob_0} $out/blobs/sha256-b559938ab7a0392fc9ea9675b82280f2a15669ec3e0e0fc491c9cb0a7681cf94
      ln -s ${blob_1} $out/blobs/sha256-f023d1ce0e55d0dcdeaf70ad81555c2a20822ed607a7abd8de3c3131360f5f0a
      ln -s ${blob_2} $out/blobs/sha256-43070e2d4e532684de521b885f385d0841030efa2b1a20bafb76133a5e1379c1
      ln -s ${blob_3} $out/blobs/sha256-ed11eda7790d05b49395598a42b155812b17e263214292f7b87d15e14003d337
      ln -s ${blob_4} $out/blobs/sha256-65d37de20e5951c7434ad4230c51a4d5be99b8cb7407d2135074d82c40b44b45

      # Move manifest file to the appropriate directory.
      mkdir -p $out/manifests/registry.ollama.ai/library/mistral-nemo
      ln -s ${manifestFile} $out/manifests/registry.ollama.ai/library/mistral-nemo/latest
    '';
  }

