# passfuse

passfuse mounts a passwordstore secret into a file.

```
Usage of ./passfuse:
  ./passfuse MOUNTPOINT pass-name
  -layout string
        Layout specifier. %p for password, %o for otp (default "%p")
```

Requirements:
  * [pass](https://www.passwordstore.org/)
  * [pass-otp](https://github.com/tadfisher/pass-otp)
