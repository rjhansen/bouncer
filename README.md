# Bouncer

A common problem in text-based online games is external resources like wikis fall
out of sync with the current player base.  This is really annoying, as nobody likes
out-of-date information.  `bouncer` helps solve this problem by making it easy to
determine which wiki pages need to be shown the door.

## Building

1. `go get github.com/rjhansen/bouncer`
2. Copy `github.com/rjhansen/bouncer/bouncer.json` to `$HOME/.bouncer.json`
3. Edit `$HOME/.bouncer.json` as appropriate for your game.
4. `go build github.com/rjhansen/bouncer`
5. In your golang installation's `bin` directory you'll find `bouncer`
6. Run and enjoy.

## The config file

 * **host**: the internet site hosting the game
 * **port**: the port on which to access the game
 * **login**: what avatar this bot should log in as
 * **password**: the password for this avatar
 * **on_connect**: what to send immediately after connecting
 * **on_disconnect**: what to send immediately before disconnecting
 * **wiki_base**: the base URL of the game's wiki
 * **active_character_page**: the page, relative to `wiki_base`, that lists all active characters
 * **active_character_regex**: the regular expression used to extract character wiki pages
 * **on_mush_as_regex**: the regular expression used to extract game names from character wiki pages
 * **finger_command**: the game's equivalent of the `+finger` command
 * **finger_regex**: the regular expression used to determine if data from the game begins a `+finger` profile
 * **recent_login_regex**: the regular expression used to determine if a player is active
