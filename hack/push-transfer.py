#!/usr/bin/env -S uv run --script
#
# /// script
# dependencies = [
#   "minio",
# ]
# ///
import argparse
import os
import shutil
import sys
import tempfile
import zipfile
from pathlib import Path
import subprocess

from minio import Minio


def run(*args, cwd=None):
    print(f"Running: {' '.join(args)}", file=sys.stderr)
    subprocess.check_call(args, cwd=cwd)


def sparse_clone(repo_url: str, branch: str, dir_path: str, dest: Path) -> None:
    """
    Perform a partial, shallow, sparse clone of a single directory using git CLI.
    """
    dest.mkdir(parents=True, exist_ok=True)

    # Initialize repo and add origin
    run("git", "init", "--quiet", str(dest))

    # Enable sparse-checkout and partial clone
    run("git", "-C", str(dest), "remote", "add", "origin", repo_url)
    run("git", "-C", str(dest), "config", "core.sparseCheckout", "true")
    run(
        "git",
        "-C",
        str(dest),
        "config",
        "remote.origin.fetch",
        f"+refs/heads/{branch}:refs/remotes/origin/{branch}",
    )
    run("git", "-C", str(dest), "config", "fetch.recurseSubmodules", "false")

    # Use modern partial-clone filter (Git >=2.19)
    run("git", "-C", str(dest), "config", "transfer.fsckobjects", "true")
    run("git", "-C", str(dest), "config", "fetch.fsckobjects", "true")

    # Write sparse-checkout patterns
    info_dir = dest / ".git" / "info"
    info_dir.mkdir(exist_ok=True)
    sparse_file = info_dir / "sparse-checkout"
    sparse_file.write_text(f"{dir_path}/*\n")

    # Fetch only the requested branch, with no tags, minimal history, no blobs outside of your path
    run(
        "git",
        "-C",
        str(dest),
        "fetch",
        "--no-tags",
        "--depth",
        "1",
        "--filter=blob:none",
        "origin",
        branch,
    )

    # Finally checkout the branch (populates only dir_path)
    run("git", "-C", str(dest), "checkout", "--quiet", branch)


def zip_folder(folder: Path, zip_path: Path) -> None:
    """Zip entire folder to zip_path."""
    print(f"Zipping folder {folder} to {zip_path}", file=sys.stderr)
    with zipfile.ZipFile(zip_path, "w", compression=zipfile.ZIP_DEFLATED) as zf:
        for root, dirs, files in os.walk(folder):
            for fname in files:
                fpath = Path(root) / fname
                zf.write(fpath, fpath.relative_to(folder.parent))


def main(
    repo_url: str,
    dir_path: str,
    branch: str,
    minio_endpoint: str,
    access_key: str,
    secret_key: str,
    minio_target: str,
) -> None:
    # Parse bucket and object path
    bucket, object_name = minio_target.split("/", 1)

    tmpdir = Path(tempfile.mkdtemp(prefix="push-images-"))
    staging = tmpdir / "staging"
    print(f"Created temp dirs: {tmpdir}, {staging}", file=sys.stderr)

    try:
        # Clone only the needed subdir
        sparse_clone(repo_url, branch, dir_path, staging)

        # Zip that subdir
        zip_path = tmpdir / "archive.zip"
        zip_folder(staging / dir_path, zip_path)

        # Upload to MinIO
        client = Minio(
            minio_endpoint, access_key=access_key, secret_key=secret_key, secure=False
        )
        print(f"Uploading {zip_path} to {bucket}/{object_name}", file=sys.stderr)
        client.fput_object(bucket, object_name, str(zip_path))
        print("Upload complete.", file=sys.stderr)

    except Exception as e:
        print(f"Error: {e}", file=sys.stderr)
        sys.exit(1)

    finally:
        print(f"Cleaning up {tmpdir}", file=sys.stderr)
        shutil.rmtree(tmpdir)


if __name__ == "__main__":
    parser = argparse.ArgumentParser(
        description="Sparse-clone with git CLI, zip, and upload to MinIO"
    )
    parser.add_argument(
        "--repo-url",
        help="Git repo URL",
        default="https://github.com/artefactual/archivematica-sampledata.git",
    )
    parser.add_argument("--branch", help="Branch to fetch", default="master")
    parser.add_argument(
        "--dir-path",
        help="Path within repo to sparse-checkout",
        default="SampleTransfers/Images/pictures",
    )
    parser.add_argument(
        "--minio-endpoint", help="MinIO server endpoint", default="localhost:7460"
    )
    parser.add_argument("--access-key", help="MinIO access key", default="minio")
    parser.add_argument("--secret-key", help="MinIO secret key", default="minio123")
    parser.add_argument(
        "--minio-target",
        help="MinIO target bucket/object",
        default="sips/transfer.zip",
    )
    args = parser.parse_args()

    main(
        repo_url=args.repo_url,
        dir_path=args.dir_path,
        branch=args.branch,
        minio_endpoint=args.minio_endpoint,
        access_key=args.access_key,
        secret_key=args.secret_key,
        minio_target=args.minio_target,
    )
