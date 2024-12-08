{ pkgs ? import <nixpkgs> {} }:

let
  # List of blob files with URLs and corresponding hashes.
  blob_0 = pkgs.fetchurl {
    curlOptsList = ["-L" "-H" "Accept:application/octet-stream"];
    url = "https://registry.ollama.ai/v2/library/mistral-nemo/blobs/sha256:970aa74c0a90ef7482477cf803618e776e173c007bf957f635f1015bfcfef0e6";
    hash = "sha256-lwqnTAqQ73SCR3z4A2GOd24XPAB7+Vf2NfEBW/z+8OY=";
  };
  blob_1 = pkgs.fetchurl {
    curlOptsList = ["-L" "-H" "Accept:application/octet-stream"];
    url = "https://registry.ollama.ai/v2/library/mistral-nemo/blobs/sha256:c71d239df91726fc519c6eb72d318ec65820627232b2f796219e87dcf35d0ab4";
    hash = "sha256-xx0jnfkXJvxRnG63LTGOxlggYnIysveWIZ6H3PNdCrQ=";
  };
  blob_2 = pkgs.fetchurl {
    curlOptsList = ["-L" "-H" "Accept:application/octet-stream"];
    url = "https://registry.ollama.ai/v2/library/mistral-nemo/blobs/sha256:ce4a164fc04605703b485251fe9f1a181688ba0eb6badb80cc6335c0de17ca0d";
    hash = "sha256-zkoWT8BGBXA7SFJR/p8aGBaIug62utuAzGM1wN4Xyg0=";
  };
  blob_3 = pkgs.fetchurl {
    curlOptsList = ["-L" "-H" "Accept:application/octet-stream"];
    url = "https://registry.ollama.ai/v2/library/mistral-nemo/blobs/sha256:31df23ea7daa448f9ccdbbcecce6c14689c8552222b80defd3830707c0139d4f";
    hash = "sha256-Md8j6n2qRI+czbvOzObBRonIVSIiuA3v04MHB8ATnU8=";
  };

  # Fetch the manifest file.
  manifestFile = pkgs.fetchurl {
    curlOptsList = ["-L" "-H" "Accept:application/octet-stream"];
    url = "https://registry.ollama.ai/v2/library/nomic-embed-text/manifests/latest";
    hash = "sha256-X5gVxKGC62iX5shIjScbZLGwaauHCj4b5yDej6F95KY=";
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
      ln -s ${blob_0} $out/blobs/sha256-970aa74c0a90ef7482477cf803618e776e173c007bf957f635f1015bfcfef0e6
      ln -s ${blob_1} $out/blobs/sha256-c71d239df91726fc519c6eb72d318ec65820627232b2f796219e87dcf35d0ab4
      ln -s ${blob_2} $out/blobs/sha256-ce4a164fc04605703b485251fe9f1a181688ba0eb6badb80cc6335c0de17ca0d
      ln -s ${blob_3} $out/blobs/sha256-31df23ea7daa448f9ccdbbcecce6c14689c8552222b80defd3830707c0139d4f

      # Move manifest file to the appropriate directory.
      mkdir -p $out/manifests/registry.ollama.ai/library/nomic-embed-text
      ln -s ${manifestFile} $out/manifests/registry.ollama.ai/library/nomic-embed-text/latest
    '';
  }

