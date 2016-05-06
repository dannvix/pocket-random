Pocket-Random
=============
Too many saved items in Pocket?
Randomly pick up something to read ... I'm feeling lucky!!

Screenshot
----------
![pocket-random screnshot](https://raw.github.com/dannvix/pocket-random/master/docs/screenshot.png)


Dependencies
------------
* Golang (we use `go1.3`)
* Golang packages
  - [`github.com/fatih/color`](https://github.com/fatih/color)
  - [`github.com/toqueteos/webbrowser`](https://github.com/toqueteos/webbrowser)
  - [`golang.org/x/crypto/ssh/terminal`](https://golang.org/x/crypto/ssh/terminal)


Usage
-----
1. Create your own app on [Pocket Developer Site](https://getpocket.com/developer/apps/)
2. Follow the script instruction to enter API keys and finish OAuth authorization
3. Enjoy reading!!


Privacy
-------
Authorized Pocket credential is saved in `~/.pocketrandom` (no password stored).

Below shows an example of `~/.pocketrandom`.
```js
{
    "username": "dannvix"
    "api_key": "42313-29d3g198f7c3da1d6715ca9d3",
    "user_code": "92d6bac3-1234-79ad-b3ca-f5c36d",
    "user_token": "0809abcd-4426-9a4b-5566-17d349",
}
```


Note
----
Implementation in Python has been deprecated.


[MIT License](http://opensource.org/licenses/mit-license.php)
-------------------------------------------------------------
Copyright (c) 2016 Shao-Chung Chen

Permission is hereby granted, free of charge, to any person obtaining a copy of this software and associated documentation files (the "Software"), to deal in the Software without restriction, including without limitation the rights to use, copy, modify, merge, publish, distribute, sublicense, and/or sell copies of the Software, and to permit persons to whom the Software is furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.
