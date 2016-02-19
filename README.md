ytplay
======

ytplay is a simple program made to solve a specific problem.

ytplay reads in lines of YouTube URLs from stdin and plays their audio one by
one in order, and echos them to stdout.

Let's say you're browsing YouTube and finding songs to listen to.  Start running
ytplay in a terminal and redirect its output to a file.  Whenever you come
across a song you want to listen to, simply paste its URL into the terminal and
it will be queued for playback.  Its URL will also be echoed into whichever file
you set up, for you to do whatever with later.

ytplay requires youtube-dl and mpv.
