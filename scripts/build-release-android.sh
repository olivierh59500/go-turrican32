#!/bin/sh
set -eu

usage() {
    echo "Usage: $0 [--unsigned-only] [--keystore PATH] [--alias NAME] [--password-file PATH]" >&2
    echo "Build the ARM64 release without installing it; signed outputs use .local/release/." >&2
    echo "Without a password file, apksigner asks for the password in your terminal." >&2
}

keystore_path=${MALAKH_ANDROID_KEYSTORE:-"$HOME/.android-keys/malakh-release.p12"}
key_alias=${MALAKH_ANDROID_KEY_ALIAS:-malakh-release}
password_file=${MALAKH_ANDROID_PASSWORD_FILE:-}
unsigned_only=false
while [ "$#" -gt 0 ]; do
    case "$1" in
        --unsigned-only) unsigned_only=true; shift ;;
        --keystore|--alias|--password-file)
            if [ "$#" -lt 2 ]; then usage; exit 2; fi
            case "$1" in
                --keystore) keystore_path=$2 ;;
                --alias) key_alias=$2 ;;
                --password-file) password_file=$2 ;;
            esac
            shift 2
            ;;
        --help|-h) usage; exit 0 ;;
        *) usage; exit 2 ;;
    esac
done

project_root=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
android_sdk=${ANDROID_HOME:-${ANDROID_SDK_ROOT:-/opt/homebrew/share/android-commandlinetools}}
java_path=${JAVA_HOME:-/opt/homebrew/opt/openjdk@17}
release_tools="$android_sdk/build-tools/36.0.0"
if [ ! -x "$java_path/bin/java" ] || [ ! -x "$release_tools/apksigner" ] || [ ! -x "$release_tools/zipalign" ]; then
    echo "Java 17 and Android Build Tools 36.0.0 are required." >&2
    exit 1
fi
if ! "$unsigned_only"; then
    if [ ! -f "$keystore_path" ]; then echo "Signing keystore was not found." >&2; exit 1; fi
    if [ -n "$password_file" ] && [ ! -r "$password_file" ]; then echo "Password file is not readable." >&2; exit 1; fi
    if [ -z "$password_file" ] && [ ! -t 0 ]; then
        echo "Use an interactive terminal for the password, or supply a private --password-file." >&2
        exit 1
    fi
fi

export JAVA_HOME="$java_path"
export ANDROID_HOME="$android_sdk"
export ANDROID_SDK_ROOT="$android_sdk"
export PATH="$java_path/bin:$PATH"
export GOWORK=off
cd "$project_root"

# Regenerate the Go library through the existing pinned mobile build. This
# preparation also refreshes the debug APK but never installs it on a device.
./scripts/run-android.sh --build-only
./android/gradlew -p android --console=plain :app:assembleRelease

version_name=$(sed -n 's/^[[:space:]]*versionName "\([^"]*\)".*/\1/p' android/app/build.gradle)
case "$version_name" in
    ""|*[!0-9A-Za-z._-]*) echo "Invalid release version name." >&2; exit 1 ;;
esac
release_dir="$project_root/.local/release"
mkdir -p "$release_dir"
source_apk="$project_root/android/app/build/outputs/apk/release/app-release-unsigned.apk"
aligned_apk="$release_dir/turrican32-$version_name-arm64.unsigned.apk"
signed_apk="$release_dir/turrican32-$version_name-arm64.apk"
"$release_tools/zipalign" -f -P 16 4 "$source_apk" "$aligned_apk"
"$release_tools/zipalign" -c -P 16 4 "$aligned_apk"
if "$unsigned_only"; then
    echo "Unsigned release for validation only: $aligned_apk"
    echo "This file must not be published or installed before signing."
    exit 0
fi

staged_apk="$release_dir/.turrican32-signing.apk"
trap 'rm -f "$staged_apk"' EXIT HUP INT TERM
set -- --ks "$keystore_path" --ks-key-alias "$key_alias"
if [ -n "$password_file" ]; then
    set -- "$@" --ks-pass "file:$password_file" --key-pass "file:$password_file"
fi
"$release_tools/apksigner" sign "$@" --out "$staged_apk" "$aligned_apk"
"$release_tools/apksigner" verify --verbose --print-certs "$staged_apk"
"$release_tools/zipalign" -c -P 16 4 "$staged_apk"
mv -f "$staged_apk" "$signed_apk"
shasum -a 256 "$signed_apk" > "$signed_apk.sha256"
echo "Signed release APK: $signed_apk"
