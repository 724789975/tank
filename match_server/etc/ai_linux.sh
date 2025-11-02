pwd
cd ../build_ai/
echo $@

nohup ./tank.x86_64 $@ &

cd -
exit

