App({
  onLaunch() {
    const settings = wx.getStorageSync("lorelattice_settings");
    if (!settings) {
      wx.setStorageSync("lorelattice_settings", {
        baseUrl: "http://localhost:8080",
        apiKey: "",
        selectedKnowledgeBaseId: ""
      });
    }
  }
});
