# Future ideas
## Add in function signature for attach points
`/lib/modules/5.19.0-40-generic/build/include`
Could essentially run a grep in that directory. If we find a match we can put it in the UI
Could be lots of duplicates so it might be tricky

## Add view for btf
`sudo bpftool btf`

## Add owner process support for socketfilter
Have to search /proc for a pid with a bpf program open that has prog_type 1.
Should also have a socket fd open. Not sure how feasible that is

## Add profile metrics for bpf programs that support it
I couldn't get bpftool to work when trying to do this. I kept getting `Error: failed to create event cycles on cpu 0`
I ran the command `sudo ~/ext-repos/bpftool/src/bpftool prog profile id 28426 duration 10 cycles instructions llc_misses`

## Add loading indicator on program start or other potentially slow events
Could look into something fancy like [this](https://github.com/navidys/tvxwidgets)

## Add a better way to search for map entries
Maps are just key value pairs. For most maps we should be able to search for a key and get the value back.
The tricky thing is we essentially just have a key and value that in some cases can be any size
Btf information might help with this but that is tricky. Need to experiment more

## Add global error message modal
I want to have a modal pop up when an action is attempted that fails. This would be a global modal that would be used for all errors. When the action fails nothing about the UI should change except for the modal displaying