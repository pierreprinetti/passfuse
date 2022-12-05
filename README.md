# passfuse

passfuse mounts a passwordstore secret into a file.

```plaintext
Usage of ./passfuse:
  ./passfuse MOUNTPOINT pass-name
  -layout string
        Layout specifier. %p for password, %o for otp (default "%p")
```

Example:

```plaintext
go install github.com/pierreprinetti/passfuse
touch secret.txt
passfuse secret.txt email/myaddress
cat secret.txt
```

Requirements:
  * [pass](https://www.passwordstore.org/)
  * [pass-otp](https://github.com/tadfisher/pass-otp)
