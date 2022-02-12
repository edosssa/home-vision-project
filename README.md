# HomeVision Take Home Project

## Getting Started

To run this program you need to have go installed on your machine.


```bash
go mod download
go run main.go
```

> The program accepts ```--pageCount``` and ```--saveDir``` flags to set the page count and directory to save the downloaded images to. If unspecified, they default to ```10``` and ```./out``` respectively.

## Gotchas

Sometimes, some of the photo urls return the following:

```
<HTML><HEAD>
<TITLE>Access Denied</TITLE>
</HEAD><BODY>
<H1>Access Denied</H1>
 
You don't have permission to access "http&#58;&#47;&#47;media&#45;cdn&#46;tripadvisor&#46;com&#47;media&#47;photo&#45;s&#47;09&#47;7c&#47;a2&#47;1f&#47;patagonia&#45;hostel&#46;jpg" on this server.<P>
Reference&#32;&#35;18&#46;2f367a5c&#46;1644683120&#46;10345591
</BODY>
</HTML>
```

My guess is that it's a rate limiting issue but I'm not entirely sure.

<p align="center">Made with 💙 by Kelvin</p>
