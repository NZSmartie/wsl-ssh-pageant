# [wsl-ssh-pageant](https://github.com/benpye/wsl-ssh-pageant) ported to Go

**Now supports multiple ssh connections concurrently!**

A Pageant bridge for WSL, enabling ssh-ageants to talk to to PuTTY Pagent or GnuPG for Windows 

![Demo](demo.gif?raw=True)

## How to use

1. On the Windows side run Pageant (or compatible agent such as gpg4win).

2. Ensure that the directory containing `wsl-ssh-pageant.exe` is on the `PATH` in WSL, for example my path contains `/mnt/c/git/wsl-ssh-pageant'

3. In WSL run the following

```
$ socat UNIX-LISTEN:/tmp/wsl-ssh-pageant.socket,unlink-close,unlink-early,fork EXEC:"wsl-ssh-pageant.exe" &
$ export SSH_AUTH_SOCK=/tmp/wsl-ssh-pageant.socket
```

4. The SSH keys from Pageant should now be usable by `ssh`!

## Credit

Thanks to 
 - [Ben Pye](https://githib.com/benpye) for his initial work on the C# version of [wsl-ssh-pageant](https://github.com/benpye/wsl-ssh-pageant)
 - [John Starks](https://github.com/jstarks/) for [npiperelay](https://github.com/jstarks/npiperelay/)
