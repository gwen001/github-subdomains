# github-subdomains

Find subdomains on GitHub.


# Warning

This repository is not public yet.
If you get access it means that you're part of the GitHub sponsors program.
For now, you're not allowed to release it, publish it or give access to anyone else.

Please remember that this program is still under development, bugs and changes may occur.


# Install

```
go get -u github.com/gwen001/github-subdomains
```

or

```
git clone https://github.com/gwen001/github-subdomains
cd github-subdomains
go install
```


# Usage

```
github-subdomains -h

Usage of github-subdomains:
  -d string
    	domain you are looking for (required)
  -e	extended mode, also look for <dummy>example.com
  -k	exit the program when all tokens have been disabled
  -o string
    	output file, default: <domain>.txt
  -raw
    	raw output
  -t string
    	github token (required), can be:
    	  • a single token
    	  • a list of tokens separated by comma
    	  • a file containing 1 token per line
    	if the options is not provided, the environment variable GITHUB_TOKEN is readed, it can be:
    	  • a single token
    	  • a list of tokens separated by comma
```

If you want to use multiple tokens, you better create a `.tokens` file in the executable directory with 1 token per line  
```
token1
token2
...
```
or use an environment variable with tokens separated by comma:  
```
export GITHUB_TOKEN=token1,token2...
```

Tokens are disabled when GitHub raises a rate limit alert, however they are re-enable 1mn later.
You can disable that feature by using the option `-k`.

<img src="https://github.com/gwen001/github-subdomains/raw/master/preview.png">


# Todo

- change the order of the extra searches ?
- ?


# Changelog

**25/09/2020**
- quick mode added
- tokens can be read from any file

**23/09/2020**
- fixed an issue in the api call (params name)
- added binary

**12/08/2020**
- improved clean function

**06/08/2020**
- max_page set forced to 10 to save 1 request for every search
- new banner (easier to maintain)
- removed `_` from the regexps
- extended regexp fixed
- improved cleaning function

**05/08/2020**
- added an option to exit the program when all tokens have been disabled instead of waiting
- rate limited tokens are disabled for 1mn then re-enabled
- removed options for languages and noise
- better page management
- panic errors handled

**04/08/2020**
- moved default languages and noise to source code
- added an option for languages and noise
- bug fixed in searches with language and noise (empty keyword)
- added search signature to avoid duplicate searches with noise
- file loading rewritten
- preview image added

**03/08/2020**
- fixed delay changed 100 -> 200
- removed useless debug messages


---

Feel free to ping me on Twitter if you have any problem to use the script.  
https://twitter.com/gwendallecoguic
