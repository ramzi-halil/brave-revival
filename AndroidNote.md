Note that on Android, [apps ignore user-installed CA by default][1]. Unless you have rooted your device, you will have to patch the APK with the following steps.

[1]: https://android-developers.googleblog.com/2016/07/changes-to-trusted-certificate.html

## Preparation

1. Obtain the XAPK somewhere.

2. Install the rebuild tools:

    ```sh
    # ubuntu for example
    sudo apt install apktool apksigner zipalign
    ```

3. Check the `aapt2` version:

    ```sh
    aapt2 version
    ```

    If `aapt2` is missing, or the version is not above `2.19-8678579` (`2.19-debian` does not count), download the prebuilt `aapt2` tool from <https://github.com/iBotPeaches/Apktool/tree/main/brut.apktool/apktool-lib/src/main/resources/prebuilt>.

## Decompile

4. Unzip the XAPK file.

    ```sh
    mkdir xapk
    cd xapk
    unzip ../*.xapk
    ```

5. Decompile the main APK

    ```sh
    apktool d -s jp.enish.shingekibo.apk
    ```

## Allow the APK to trust user-defined certificate

6. Put the following content into `res/xml/network_security_config.xml`

    ```xml
    <?xml version="1.0" encoding="utf-8"?>
    <network-security-config>
        <base-config>
            <trust-anchors>
                <certificates src="user"/>
                <certificates src="system"/>
            </trust-anchors>
        </base-config>
    </network-security-config>
    ```

7. Edit `AndroidManifest.xml`, add the property `android:networkSecurityConfig="@xml/network_security_config"` to the `<application>` tag.

# Rebuild and sign

8. Rebuild the APK

    ```sh
    PATH=..:"$PATH" apktool b --use-aapt2 jp.enish.shingekibo
    rm jp.enish.shingekibo.apk
    zipalign 4 jp.enish.shingekibo/dist/jp.enish.shingekibo.apk jp.enish.shingekibo.apk
    ```

9. Generate signing key

    ```sh
    keytool -genkeypair -keystore aotbo.p12 -keyalg RSA -keysize 2048 -validity 10000 -alias aotbo -storepass 123456 -keypass 123456 -dname 'CN=jp.enish.shingekibo' -noprompt
    ```

10. Sign *all* APKs

    ```sh
    apksigner sign --ks aotbo.p12 --ks-key-alias aotbo --ks-pass pass:123456 --key-pass pass:123456 jp.enish.shingekibo.apk
    apksigner sign --ks aotbo.p12 --ks-key-alias aotbo --ks-pass pass:123456 --key-pass pass:123456 UnityDataAssetPack.apk
    apksigner sign --ks aotbo.p12 --ks-key-alias aotbo --ks-pass pass:123456 --key-pass pass:123456 config.*.apk
    ```

11. Rebuild the XAPK

    ```sh
    zip -0 patched.xapk jp.enish.shingekibo.apk icon.png UnityDataAssetPack.apk config.*.apk manifest.json
    ```
