using System;
using System.Collections.Generic;
using System.Linq;
using System.Net.Http;
using System.Text;
using System.Threading.Tasks;
using UnityEngine;

public class AsyncWebRequest
{
	/// <summary>
	/// 异步GET请求
	/// </summary>
	/// <param name="url"></param>
	/// <param name="requestComplete"></param>
	/// <returns></returns>
	public async void Get(string url, Action<bool, byte[]> requestComplete)
	{
		var result = await AsyncGetRequest(url);
		requestComplete?.Invoke(result.Item1, result.Item2);
	}

	private static async Task<(bool, byte[])> AsyncGetRequest(string url)
	{
		var tcs = new TaskCompletionSource<(bool, byte[])>();
		HttpClient client = new HttpClient();
		try
		{
			// 发送GET请求获取响应
			HttpResponseMessage response = await client.GetAsync(url);
			// 如果响应成功
			if (response.IsSuccessStatusCode)
			{
				// 读取响应体并设置结果为true和响应体
				byte[] responseBody = await response.Content.ReadAsByteArrayAsync();
				tcs.SetResult((true, responseBody));
			}
			// 如果响应失败
			else
			{
				// 设置结果为false和响应原因
				tcs.SetResult((false, null));
				Debug.LogError("请求失败");
			}
		}
		// 捕获所有异常并设置结果为false和异常消息
		catch (Exception ex)
		{
			Debug.LogError(ex);
			tcs.SetResult((false, null));
		}
		// 不管是否发生异常，最后都释放 HttpClient 的资源
		finally
		{
			client.Dispose();
		}
		// 返回异步任务的结果
		return await tcs.Task;
	}

	/// <summary>
	/// 异步POST请求
	/// </summary>
	/// <param name="url"></param>
	/// <param name="postData"></param>
	/// <param name="customHeaders"></param>
	/// <param name="requestComplete"></param>
	public async void Post(string url, string postData, Dictionary<string, string> customHeaders, Action<bool, byte[]> requestComplete = null)
	{
		var result = await AsyncPostRequest(url, postData, customHeaders);
		requestComplete?.Invoke(result.Item1, result.Item2);
	}

	private async Task<(bool, byte[])> AsyncPostRequest(string url, string postData, Dictionary<string, string> requestHeaders)
	{
		var tcs = new TaskCompletionSource<(bool, byte[])>();
		using (HttpClient client = new HttpClient())
		{
			try
			{
				// 创建StringContent对象时，显式设置Content-Type
				StringContent content = new StringContent(postData, Encoding.UTF8, "application/json");

				// 添加额外的请求头，但排除Content-Type
				if (requestHeaders != null)
				{
					foreach (var header in requestHeaders.Where(h => h.Key.ToLower() != "content-type"))
					{
						client.DefaultRequestHeaders.Add(header.Key, header.Value);
					}
				}
				// 发送POST请求
				HttpResponseMessage response = await client.PostAsync(url, content);

				// 判断请求是否成功
				if (response.IsSuccessStatusCode)
				{
					var responseBody = await response.Content.ReadAsByteArrayAsync();
					tcs.SetResult((true, responseBody));
				}
				else
				{
					Debug.LogError($"请求失败,错误码: {(int)response.StatusCode}");
					tcs.SetResult((false, Encoding.UTF8.GetBytes(response.ReasonPhrase)));
				}
			}
			catch (Exception ex)
			{
				tcs.SetResult((false, Encoding.UTF8.GetBytes(ex.Message)));
			}
		}
		return await tcs.Task;
	}
}