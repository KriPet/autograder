$("#testplagiarism").click(function(event){
  return performTestPlagiarismClick("individual");
});

$("#grouptestplagiarism").click(function(event){
  return performTestPlagiarismClick("group");
});

function performTestPlagiarismClick(labs) {
  $.post("/event/manualtestplagiarism", {"course": course, "labs": labs}, function(){
    alert("The anti-plagiarism command was sent. It will take several minutes at the minimum to process. Please be patient. The results will appear in x.");
  }).fail(function(){
    alert("The anti-plagiarism command failed.");
  });
  event.preventDefault();
  return false
}

// loads anti-plagiarism results for a user's lab from server and updates html.
var loadLabApResults = function(user, lab){
  $.getJSON("/course/aplabresults", {"Labname": lab, "Course": course, "Username": user}, function(data){
    $("#mossResults").text(data.MossPct).append("%");
    $("#jplagResults").text(data.JplagPct).append("%");
    if (data.DuplPct == 0.0) {
    		$("#duplResults").text("False");
    }
    else {
    		$("#duplResults").text("True");
    }
  }).fail(function(){
    $("#mossResults").text("").append("-1% : Error");
    $("#jplagResults").text("").append("-1% : Error");
    $("#duplResults").text("").append("-1% : Error");
  });
}

// loads anti-plagiarism results for a user's lab from server and updates html.
var loadUserApResults = function(index, element){
  var username = $(element).attr("id");
  $.getJSON("/course/apuserresults", {"Course": course, "Username": username}, function(data){
    $.each(data, function(labname, s){
      if(labname == "") {
        return
      }
      var count = 0;
      
      if (s.MossPct > 0.0) {
        count++;
      }
      if (s.JPlagPct > 0.0) {
        count++;
      }
      if (s.DuplPct > 0.0) {
        count++;
      }      	

      $("tr#" + data.User + " > td." + labname).text(count);
    });
  }).fail(function(){

  });
}