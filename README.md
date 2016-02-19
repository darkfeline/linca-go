linca
=====

linca is a simple directory watching and linking program.

Run linca with arguments supplying watch and destination directories:

    $ linca a b

All files and directories created in or moved to `a` will be linked to `b`.
Specifically, on inotify events CREATE and MOVED_TO, linca will link files and
`cp -al` directories to `b`.

linca is useful for, e.g., processing downloaded files from torrent clients.

linca requires inotify-tools.
