using Google.Protobuf.WellKnownTypes;
using System;
using System.Collections;
using System.Collections.Generic;
using System.Diagnostics;
using System.Reflection;
using UnityEngine;

public class WSMsgProcess : Singleton<WSMsgProcess>
{
	public void ProcessMessage(object sender, Any msg)
	{
		string name = Any.GetTypeName(msg.TypeUrl);
		//UnityEngine.Debug.Log($"ProcessMessage: {name}, {pConnector}");
		if (handlerDict.ContainsKey(name))
		{
			var method = handlerDict[name];
			method(sender, msg);
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
			var attr = method.GetCustomAttribute<WSHandlerAttribute>();
			if (attr != null)
			{
				var name = attr.Name;
				if (handlerDict.ContainsKey(name))
				{
					UnityEngine.Debug.Assert(false, "Duplicate RpcHandlerAttribute name: " + name);
				}
				else
				{
					handlerDict[name] = delegate (object sender, Any msg)
					{
						method.Invoke(null, new object[] { sender, msg });
					};
				}
			}
		}
	}

	delegate void MsgHandler(object sender, Any msg);

	private Dictionary<string, MsgHandler> handlerDict = new Dictionary<string, MsgHandler>();

}
