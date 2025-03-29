#!/bin/sh
# Should run from project root dir

sh po/update_potfiles.sh

cat ./po/POTFILES | xargs xgettext --language=C --keyword=T_ --keyword=TN_:1,2 --keyword=TD_:2 --keyword=TC_:1c,2 -o po/installer.pot --from-code=UTF-8 --add-comments --package-name=apm
