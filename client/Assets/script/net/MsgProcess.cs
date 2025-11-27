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
				UnityEngine.Debug.LogError($"Error processing message: {name}\n{e.Message}\n{e.StackTrace}");
			}
		}
		else
		{
			UnityEngine.Debug.LogError($"No handler for message: {name}");
		}
	}

	public void RegisterHandler(object handler)
	{
		foreach (var method in handler.GetType().GetMethods(BindingFlags.NonPublic | BindingFlags.Static))
		{
			var attr = method.GetCustomAttribute<RpcHandlerAttribute>();
			if (attr != null)
			{
				var name = attr.Name;
				if (handlerDict.ContainsKey(name))
				{
					UnityEngine.Debug.LogError("Duplicate RpcHandlerAttribute name: " + name);
				}
				else
				{
					handlerDict[name] = delegate (object pConnector, Any msg) {
						method.Invoke(null, new object[] { pConnector, msg });
					};
				}
			}
		}
	}

	delegate void MsgHandler(object pConnector, Any msg);

	private Dictionary<string, MsgHandler> handlerDict = new Dictionary<string, MsgHandler>();

}
