kubectl -n tank get svc -l auto-clean=true -o jsonpath='{range .items[*]}{.metadata.name}{" "}{.metadata.labels.app}{"\n"}{end}' | while read svc job; do
if ! kubectl get job $job -n tank ; then
	echo "Deleting orphaned service: $svc"
	kubectl delete svc $svc -n tank
fi
done
