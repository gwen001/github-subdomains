# github-subdomains

Find subdomains on GitHub.


# Warning

This repository is not public yet.
If you get access it means that you're part of the GitHub sponsors program.
The tool will be released as soon as the current goal will be reached.
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
    	github token (required)
```

If you want to use multiple tokens, you should create a `.tokens` file in the executable directory with 1 token per line  
```
token1
token2
...
```
or use an environment variable with tokens separated by comma:  
```
export GITHUB_TOKEN=token1,token2...
```

<img src="https://github.com/gwen001/github-subdomains/raw/master/preview.png">


# Todo

- improve cleanSubdomain function
- change the order of the extra searches ?
- ?


# Changelog

**06/08/2020**
- new banner (easier to maintain)  
- removed `_` from the regexp  
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
