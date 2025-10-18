pwd
cd ../build_ls/
echo $@

nohup ./tank.x86_64 $@ &

cd -
exit

