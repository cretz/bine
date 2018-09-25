## HTTP Alt-Svc Example

The Tor browser now supports `Alt-Svc` headers to be onion services as
[HTTP alternate services](https://tools.ietf.org/html/rfc7838) to traditional sites.
[This Cloudflare post](https://blog.cloudflare.com/cloudflare-onion-service/) explains how they use it. This example
shows how to do it yourself. Since

Specifically, this example listens on all IPs on 80 for insecure HTTP requests. It also listens on an onion service for
insecure HTTP requests. Both services return `Alt-Svc` addresses to an onion service that is run securely.

**NOTE: Only do the steps in this example if you are comfortable with and aware of the consequences.**

### Setup

We're going to self-sign a certificate for use in the Tor browser. First, download `mkcert` and run:

    mkcert -install

This will install a fake CA on your local machine that certs can be generated from. Remember the path it says the local
CA is at or run `mkcert -CAROOT` to get it back. We will use this "CA path" later.

Now the CA must be added to the Tor browser. By default the Tor browser doesn't use the cert database so it cannot store
the overrides. To change this, go to `about:config` click through warning and change `security.nocertdb` to `false`. Now
that CAs can be added, go to `Options` (i.e. `about:preferences`) > `Privacy & Security` > `Certificates` section at the
bottom > `View Certificates...` > `Authorities` tab > `Import...` > choose `rootCA.pem` from the "CA path" from
earlier > check "Trust this CA to identify websites" and click `OK`. Restart the Tor browser.

### Running

Point yor DNS to this machine's IP (or use start `ngrok http 80` via [ngrok.com](https://ngrok.com/))

TODO: the rest...

TODO: alt-svc onions appear broken in TBB