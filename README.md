go_snappshot
============

Snappshot rewritten in Golang

Uses imagemagick, in particular this library https://github.com/gographics/imagick/

Once imagemagick devel library is installed, than get the required go packages.


Get dependencies 
 * go get github.com/gographics/imagick/imagick
 * go get github.com/robfig/revel/revel
 
This is built on revel, so use the revel commands to do what you want

To Test
  revel run github.com/justinjudd/go_snappshot

To Package
  revel package github.com/justinjudd/go_snappshot
