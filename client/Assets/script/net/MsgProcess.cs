using Google.Protobuf.WellKnownTypes;
using System;
using System.Collections;
using System.Collections.Generic;
using System.Diagnostics;
using System.Reflection;
using UnityEngine;

public class MsgProcess : Singleton<MsgProcess>
{
	public void ProcessMessage(IntPtr pConnector, Any msg)
	{
		string name = Any.GetTypeName(msg.TypeUrl);
		//UnityEngine.Debug.Log($"ProcessMessage: {name}, {pConnector}");
		if (handlerDict.ContainsKey(name))
		{
			var method = handlerDict[name];
			method(pConnector, msg);
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
					UnityEngine.Debug.Assert(false, "Duplicate RpcHandlerAttribute name: " + name);
				}
				else
				{
					handlerDict[name] = delegate (IntPtr pConnector, Any msg) {
						method.Invoke(null, new object[] { pConnector, msg });
					};
				}
			}
		}
	}

	delegate void MsgHandler(IntPtr pConnector, Any msg);

	private Dictionary<string, MsgHandler> handlerDict = new Dictionary<string, MsgHandler>();

}
