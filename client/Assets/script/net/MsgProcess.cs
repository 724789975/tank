using Google.Protobuf.WellKnownTypes;
using System;
using System.Collections;
using System.Collections.Generic;
using System.Diagnostics;
using System.Reflection;
using UnityEngine;

public class MsgProcess : Singleton<MsgProcess>
{
	public void ProcessMessage(object pConnector, Any msg)
	{
		string name = Any.GetTypeName(msg.TypeUrl);
		if (handlerDict.ContainsKey(name))
		{
			var method = handlerDict[name];
			try
			{
				method(pConnector, msg);
			}
			catch (Exception e)
			{
				UnityEngine.Debug.LogError($"[Msg][ProcessMessage] handle message failed, name={name} connector={pConnector}, exception={e.Message}\n{e.StackTrace}");
			}
		}
		else
		{
			UnityEngine.Debug.LogError($"[Msg][ProcessMessage] no handler for message: {name}, connector={pConnector}");
		}
	}

	public void RegisterHandler(System.Type handler)
	{
		int registered = 0;
		foreach (var method in handler.GetMethods(BindingFlags.NonPublic | BindingFlags.Static))
		{
			var attr = method.GetCustomAttribute<RpcHandlerAttribute>();
			if (attr != null)
			{
				var name = attr.Name;
				if (handlerDict.ContainsKey(name))
				{
					UnityEngine.Debug.LogError($"[Msg][RegisterHandler] duplicate RpcHandlerAttribute name: {name} in {handler.Name}");
				}
				else
				{
					handlerDict[name] = delegate (object pConnector, Any msg) {
						method.Invoke(null, new object[] { pConnector, msg });
					};
					registered++;
				}
			}
		}
		UnityEngine.Debug.Log($"[Msg][RegisterHandler] registered {registered} handlers from {handler.Name}, total={handlerDict.Count}");
	}

	delegate void MsgHandler(object pConnector, Any msg);

	private Dictionary<string, MsgHandler> handlerDict = new Dictionary<string, MsgHandler>();

}
