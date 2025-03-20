# mailrelay - daemonless sendmail replacement for Docker environments.

Use a set of email relays to easily send emails from Docker containers without setting up a SMTP container.

You can use this from eg. your PHP environment. Set this in your `php.ini`:

```
;sendmail_path = /usr/sbin/sendmail -t -i
sendmail_path = /usr/local/bin/mailrelay
```

Set your relays using an environment variable. `mailrelay` will randomize the list and then try to relay through the list, one by one, until it either succeeds or it has no other server to try, in which case it will fail.

```
export MAILRELAY_SERVERS="relay1.domain.tld:25;relay2.domain.tld:25;relay3.domain.tld:25"
```

The email relays will need to be configured to accept email from the Docker container without authentication.

I needed this solution in a legacy environment until a full transition to background jobs.
