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
