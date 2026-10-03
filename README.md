**Brave Revival** is a private server for enabling solo play for *Attack on Titan: Brave Order* which has been shut down since 2026 September 28th.

> [!NOTE]
> This project is not affiliated with Enish in any way. It is a fan-made project intended for personal use primarily.

# Preparation

To compile the code here, you need to install [Go](https://go.dev/dl/) v1.27.1 or above.

Brave Revival is compatible with the final two versions of *Attack on Titan: Brave Order* (v1.43.274 & v1.44.274). We strongly recommend staying on v1.43.274 for best experience (the new font used by v1.44.274 does not display correctly beyond the refund dialog).

> [!TIP]
> If you are using Android, the game need to patched to allow intercepting traffic. Please prepare the original APK or XAPK file for patching, which we will explain [below](#android).

# Download assets

This server provides the API and game data ("master database"), but you have to obtain the game assets (images, 3D models, sounds, etc.) yourself. We will not share these with you.

## From Android game cache

If you had played the game before and had not cleared the cache, you can copy the downloaded assets from the User data directory in `/sdcard/Android/data/jp.enish.shingekibo/files/Assets/*`.

## From official server

If the Enish asset server is still online, you can use the `download-assets` tool to download the assets from the official server. The tool will download all assets and store them in the `raw` directory.

```sh
go run ./cmd/download-assets -p android
```
| Flag | Meaning |
|------|---------|
| `-p` | Platform of the game, either `android` or `ios`. |
| `-o` | Output directory, default to `./raw`. |
| `-i` | Input resources list, default to `./db/master/resources.csv`. |
| `-u` | Base URL of the asset server. |
| `-j` | Download concurrency, default to `2`. |
| `-f` | Filter, only download a subset of all assets with given prefix (see below). |
| `-n` | Dry run, only print total size to be downloaded. |

Commonly used prefixes for `-f` are:

| Prefix | Asset type    | Total size |
|--------|---------------|-----------:|
| 98 | Gacha movies      | 5.4 GiB    |
| 38 | Anime screenshots | 5.1 GiB    |
| 99 | Sounds            | 448 MiB    |
| 8  | Event images      | 421 MiB    |
| 1  | 3D models         | 286 MiB    |
| 7  | Character images  | 112 MiB    |
|    | (remaining)       | 469 MiB    |

To serve a single platform you need about **12 GiB** of free disk space. To serve both you need about **19 GiB** (sounds and movies are shared).

# Run

Edit `config.json` to suit your environment. In particular, edit "advertise_host" to an IP address or domain name that is reachable by your device. Usually you can reuse the output from the `ifconfig` (macOS/Linux) or `ipconfig` (Windows) command. If the target is BlueStacks or other Android simulators, keep the "advertise_host" as the default (10.0.2.2).

Run the server with
```
go run ./cmd/brave-revival config.json
```

The rest of this document assumed `advertise_host` is 10.0.2.2 and `proxy_port` is 13845.

## Firewall

If the server has strict firewall rules, please allow incoming connections from the device's IP to these ports, or reconfigure them to allowed port numbers.

| Port  | Protocol | Configuration | Purpose |
|-------|----------|---------------|---------|
| 5055  | UDP (Photon) | - | Battle (master server); **hard-coded and cannot be changed** |
| 5056  | UDP (Photon) | "photon_port" | Battle (game server) |
| 10000 | TCP (WebSocket) | "chat_port" | Chat |
| 10002 | TCP (gRPC) | "party_port" | Party and notification |
| 13845 | TCP (HTTP) | "proxy_port" | Proxy and control |

# Proxying game traffic

Brave Revival works by replacing HTTPS requests to the official Enish server with requests to your local server. For this to work, you must configure your device to use the local server as an HTTPS proxy. Since the traffic is encrypted, you also need to install a custom CA certificate on your device so that the device trusts the local server.

## Android

1. **Install CA**
    * On the device, open `http://10.0.2.2:13845/ca.crt` to download the CA certificate. Alternatively, copy `./tls-cache/ca.crt` from the computer to the device.
    * Go to *Settings* → *Security* → *More security settings* -> *Encrypt & credentials* → *Install a certificate* → *CA certificate*, click *INSTALL ANYWAY*, then choose the downloaded certificate to install.
    * If you are using BlueStacks, the *Security* section is hidden by default. You can enter this setting via `adb` by running this from the host:
        ```sh
        adb shell am start -a android.settings.SECURITY_SETTINGS
        ```

2. **Patch game to support user-installed CA**
    * On Android, apps ignore user-installed CA by default. Thus we have to patch the package to trust user-installed CA.
    * Prepare the [XAPK file](https://apkpure.com/how-to/how-to-install-xapk-apk) of the game, then run
        ```sh
        go run ./cmd/patch-xapk -o patched.xapk game.xapk
        ```
    * Alternatively, following [AndroidNote.md](AndroidNote.md) to patch the APK or XAPK manually.
    * If your device cannot install XAPK, besides installing an Installer, you can also unzip the XAPK and install the 3 split-APKs through `adb install-multiple`.

3. **Configure proxy via PAC** (for most Android devices, except BlueStacks)
    * Go to *Settings* → *Network & Internet* → *Wi-Fi* → tap the gear icon ⚙️ next to the connected Wi-Fi network → *Advanced options*
    * Under *Proxy*, select *Proxy Auto-Config*.
    * In the *PAC URL* field, enter `http://10.0.2.2:13845/proxy.pac`.
    * Click *Save*.

4. **Configure proxy via Sing-box** (only when step 3 does not work)
    * Some systems like BlueStacks do not expose PAC settings, so instead we route traffic via a VPN service.
    * Install [Sing-box](https://sing-box.sagernet.org/clients/android/).
    * On the dashboard, click the plus icon ➕ in Profiles → *Create Manually*
    * Choose *Type* = *Remote*, then in *Remote Configuration* enter `http://10.0.2.2:13845/singbox.json` as the *URL*.
    * Click *Create*.
    * Click the play button ▶️ to start the VPN service.

## iOS

1. **Install CA**
    * On the device, open `http://10.0.2.2:13845/ca.crt` in Safari to download the CA certificate.
    * Go to *Settings* → *General* → *VPN & Device Management* → *Brave Revival CA* (under *Downloaded profiles*) → *Install* → (enter device password) → *Install* → *Install* → *Done*.
    * Go to *Settings* → *General* → *About* → *Certificate Trust Settings*, then enable the toggle for *Brave Revival CA*.

2. **Configure proxy via PAC**
    * Go to *Settings* → *Wi-Fi* → tap the ⓘ icon next to the connected Wi-Fi network → *Configure Proxy* → *Automatic*.
    * In the *URL* field, enter `http://10.0.2.2:13845/proxy.pac`.

# Features

The server is designed for solo play with the focus on capturing scenes instead of game play or collecting resources.

* You start with an account with all units, skins and items obtained, level maximized (at Lv.780), VIP level maximized (at VIP Lv.12), and all titles unlocked.

* These are supported:
    * Change the furniture and items in corps room (兵団集め)
    * Visit randomly generated rooms
    * Change your name and comment
    * Change guild symbol and name
    * Enhancing any units
        * Materials are never consumed.
        * Proficiency (熟練度) and Skill level enhancement always fail by default. Enable ☑️ "Until succeed" (成功するまで) to make them always succeed. This is to allow both voices can be played.
        * Successful proficiency enhancement always result in +1 regardless of material.
    * Attach and detach medals (勲章)
    * Level-up and reset the skill board
    * Change deck (編成) content and labels
    * Watch scenarios in the main story

* Most other details are immutable. The following actions are all no-ops:
    * Receive rewards
    * Buy items from shop
    * Exchange items
    * Play roulette
    * Perform "technical training" (技術訓練)
    * Register "Titan research" (巨人研究)
    * Send "Like" to randomly generated rooms
    * Donate to the guild
    * Edit the guild message boards
    * Redraw "latent parameters" (潜在能力)
    * Rarity-up any medals
    * Lengthen "lap quests" (周回クエスト) bonus time
    * Sweep for expedition (壁外調査掃討)

* The entire server has only a single player. Thus,
    * Friend list is always empty
    * Chat rooms are empty
    * You cannot create new chat groups
    * Mercenary list is empty
    * "Like" list is empty

* You cannot lose or gain any items or units. The inventory is unchanged in-game despite some operation may show success.

* Gacha does not follow the standard probabilities, high rarity units appear much more often. Nevertheless, as described above the pulled units aren't added to inventory.

## Known issues

* **Battle does not work yet**

* Advertisement will be disabled in the future. Currently you can still watch ads but they will not give you any rewards.

* Guild facilities' levels should be maximized. Currently they are all at Lv.1.

* The Practice-GvG area currently crashes with 404.

* Guild Interception Battles (共闘迎撃戦) cannot be unlocked.

* PvP shows no opponents.

## Advanced corps room editing

On the server machine, you can go to `http://127.0.0.1:13845/agito` to arbitrarily edit the corp room.

After finish editing, click "Save" to save the changes, and then in the game either click the Stopwatch icon ⏱️ (何か変化があるかも？) on the home screen to refresh.

Unlike editing within the game, the item and visitor selection are unrestricted. For instance you can place a Sofa on top of the table, choose visitors beyond the preset line-up, or even have duplicate items or visitors.

But keep in mind:

* Table-top items have opposite orientation than other items. Other items placed in the "☕️ 1xx" area will face backward, and vice-versa.
* The base level of table-top items are at the table top height. Therefore, if you place a table-top item on the floor, the visitors will be sunken below the floor.
* "🛋️ 4xx" areas only allows large items. Visitors will not come if small items are placed in the "🛋️ 4xx" area. On the other hand, large items can be placed in any area.
* Visitors with special motion are highlighted in blue (motion #2) or purple (motion #3). Some items do not support special motion (mainly the large items and Bureo-Darumas), and those special-motion visitors will not appear.
* As usual, the controllable character is always be a different than any visitors. So it is impossible to have a scene which the entire room is filled with a unique character. At minimum you have to allow the hooded placeholder unit to appear.
