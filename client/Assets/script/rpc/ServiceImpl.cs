class ServicerImpl : TankGameService.TankGameService.TankGameServiceBase
{
	public override System.Threading.Tasks.Task<TankGameService.Pong> ping(TankGameService.Ping request, Grpc.Core.ServerCallContext context)
	{
		return System.Threading.Tasks.Task.FromResult(new TankGameService.Pong { Ts = request.Ts });
	}
}
