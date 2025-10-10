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
	/// �첽GET����
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
			// ����GET�����ȡ��Ӧ
			HttpResponseMessage response = await client.GetAsync(url);
			// �����Ӧ�ɹ�
			if (response.IsSuccessStatusCode)
			{
				// ��ȡ��Ӧ�岢���ý��Ϊtrue����Ӧ��
				byte[] responseBody = await response.Content.ReadAsByteArrayAsync();
				tcs.SetResult((true, responseBody));
			}
			// �����Ӧʧ��
			else
			{
				// ���ý��Ϊfalse����Ӧԭ��
				tcs.SetResult((false, null));
				Debug.LogError("����ʧ��");
			}
		}
		// ���������쳣�����ý��Ϊfalse���쳣��Ϣ
		catch (Exception ex)
		{
			Debug.LogError(ex);
			tcs.SetResult((false, null));
		}
		// �����Ƿ����쳣������ͷ� HttpClient ����Դ
		finally
		{
			client.Dispose();
		}
		// �����첽����Ľ��
		return await tcs.Task;
	}

	/// <summary>
	/// �첽POST����
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
				// ����StringContent����ʱ����ʽ����Content-Type
				StringContent content = new StringContent(postData, Encoding.UTF8, "application/json");

				// ��Ӷ��������ͷ�����ų�Content-Type
				if (requestHeaders != null)
				{
					foreach (var header in requestHeaders.Where(h => h.Key.ToLower() != "content-type"))
					{
						client.DefaultRequestHeaders.Add(header.Key, header.Value);
					}
				}
				// ����POST����
				HttpResponseMessage response = await client.PostAsync(url, content);

				// �ж������Ƿ�ɹ�
				if (response.IsSuccessStatusCode)
				{
					var responseBody = await response.Content.ReadAsByteArrayAsync();
					tcs.SetResult((true, responseBody));
				}
				else
				{
					Debug.LogError($"����ʧ��,������: {(int)response.StatusCode}");
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